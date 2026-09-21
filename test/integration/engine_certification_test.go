package integration

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/replay"
	"github.com/stretchr/testify/require"
)

// realEngine returns the Starfighter engine binary pinned for this checkout.
// CI builds it from the commit in engine.lock and sets AGENTRIX_REQUIRE_ENGINE=1.
func realEngine(t *testing.T) string {
	t.Helper()
	bin := os.Getenv("AGENTRIX_ENGINE_BIN")
	if bin == "" {
		bin = filepath.Join(repoRoot, "bin", "starfighter-engine")
	}
	if _, err := os.Stat(bin); err != nil {
		if os.Getenv("AGENTRIX_REQUIRE_ENGINE") == "1" {
			t.Fatalf("engine binary required: %v", err)
		}
		t.Skipf("engine binary not built (%s): run make engine", bin)
	}
	return bin
}

func decodeStaged(t *testing.T, e *env, runID string) *replay.Document {
	t.Helper()
	f, err := e.artifacts.Open("replays/staging/" + runID + ".ndjson.gz")
	require.NoError(t, err)
	defer f.Close()
	zr, err := gzip.NewReader(f)
	require.NoError(t, err)
	doc, err := replay.Decode(zr)
	require.NoError(t, err)
	return doc
}

func hashes(doc *replay.Document) []string {
	out := make([]string, len(doc.Snapshots))
	for i, s := range doc.Snapshots {
		out[i] = s.StateHash
	}
	return out
}

// The real engine plays a full competitive match through the real worker,
// and the same sealed spec re-executed in fresh processes reproduces every
// state hash and the result exactly (certification on this platform).
func TestEngine_RealMatchAndDeterminismCertification(t *testing.T) {
	bin := realEngine(t)
	e := newEnv(t, envOptions{engineBin: bin})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w, "bot_hunter.py", "bot_random.py")
	ticket := schedule(t, comp.organizer, comp.matchID)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w.Run(ctx); close(done) }()
	match := comp.organizer.waitMatch(comp.matchID, 5*time.Minute, "finished", "failed")
	cancel()
	<-done
	require.Equal(t, "finished", match["state"], "runs: %v", match["runs"])
	results := match["results"].([]any)
	require.Len(t, results, 2)

	run, err := e.store.GetRun(context.Background(), ticket["run_id"].(string))
	require.NoError(t, err)
	require.Equal(t, 60, run.ExecutionSpec.TickRate.Numerator)
	require.Equal(t, 1, run.ExecutionSpec.TickRate.Denominator)
	require.Equal(t, e.engineSHA, run.ExecutionSpec.Engine.SHA256)

	published, err := e.artifacts.Open("replays/published/" + run.ID + ".ndjson.gz")
	require.NoError(t, err)
	zr, err := gzip.NewReader(published)
	require.NoError(t, err)
	official, err := replay.Decode(zr)
	require.NoError(t, err)
	_ = published.Close()
	require.Greater(t, len(official.Snapshots), 1)
	t.Logf("official replay: %d snapshots, reason=%s winner=%s scores=%v", len(official.Snapshots), official.Result.Reason, official.Result.Winner, official.Result.Scores)

	exec := executor.NewExecutor(e.runtime, e.artifacts, executor.SubprocessEngineFactory(bin))
	for i := 0; i < 3; i++ {
		res := model.Reservation{RunID: newUUID(), MatchID: run.MatchID, Attempt: 1, WorkerID: "cert", FencingToken: 1, Spec: run.ExecutionSpec}
		completion, err := exec.Execute(context.Background(), res)
		require.NoError(t, err, "certification run %d", i)
		doc := decodeStaged(t, e, res.RunID)
		require.Equal(t, hashes(official), hashes(doc), "state hash sequence diverged on run %d", i)
		require.Equal(t, official.Result.Scores, doc.Result.Scores)
		require.Equal(t, official.Result.Winner, doc.Result.Winner)
		require.Equal(t, official.Result.Reason, doc.Result.Reason)
		require.Equal(t, official.Result.FinalStateHash, completion.FinalStateHash)
	}

	// A different seed is a different spec and must produce a different game.
	other := run.ExecutionSpec
	other.Seed++
	_, _, err = other.Seal()
	require.NoError(t, err)
	res := model.Reservation{RunID: newUUID(), MatchID: run.MatchID, Attempt: 1, WorkerID: "cert", FencingToken: 1, Spec: other}
	_, err = exec.Execute(context.Background(), res)
	require.NoError(t, err)
	require.NotEqual(t, hashes(official), hashes(decodeStaged(t, e, res.RunID)))

	// The replay is served to spectators.
	replayID := match["replay"]
	for replayID == nil || replayID.(map[string]any)["available"] != true {
		time.Sleep(100 * time.Millisecond)
		replayID = comp.organizer.expect(http.StatusOK, "GET", "/api/v1/matches/"+comp.matchID, nil)["replay"]
	}
}

// A worker whose engine binary does not match the pinned digest refuses to
// run the spec (no silent substitution of the engine).
func TestEngine_DigestMismatchIsRejected(t *testing.T) {
	bin := realEngine(t)
	e := newEnv(t, envOptions{engineBin: bin})
	e.registerWorker("w1")
	comp := e.competition(e.worker("w1"))
	schedule(t, comp.organizer, comp.matchID)
	res := e.reserveOnly("w1")
	exec := executor.NewExecutor(e.runtime, e.artifacts, executor.SubprocessEngineFactory(fakeEngineBin))
	_, err := exec.Execute(context.Background(), *res)
	var runErr *executor.RunError
	require.ErrorAs(t, err, &runErr)
	require.Equal(t, model.ErrorSpec, runErr.Class)
}

// The real engine accepts the canonical example of the protocol schema and
// rejects the invalid one.
func TestEngine_ProtocolExamples(t *testing.T) {
	bin := realEngine(t)
	for _, tc := range []struct {
		file string
		ok   bool
	}{{"valid/initialize-match.json", true}, {"invalid/initialize-match-no-players.json", false}} {
		raw, err := os.ReadFile(filepath.Join(repoRoot, "protocol/engine/v1/examples", tc.file))
		require.NoError(t, err)
		var envelope struct {
			Payload engine.InitializeMatchRequest `json:"payload"`
		}
		require.NoError(t, json.Unmarshal(raw, &envelope))
		client := engine.NewSubprocessClient()
		require.NoError(t, client.Start(context.Background(), engine.StartConfig{BinaryPath: bin, HandshakeTimeout: 10 * time.Second}))
		_, err = client.InitializeMatch(context.Background(), envelope.Payload)
		if tc.ok {
			require.NoError(t, err, tc.file)
		} else {
			require.Error(t, err, tc.file)
		}
		_ = client.Close(context.Background())
	}
}
