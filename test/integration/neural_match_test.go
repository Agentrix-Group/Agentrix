package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	replaystream "github.com/Agentrix-Group/Agentrix/src/replay"
	"github.com/stretchr/testify/require"
)

// ADR-0014 (N6): end to end on the platform. Three neural bots (.onnx and
// .npz web templates, .safetensors example) are uploaded as bundles, admitted
// by the worker on python-ml-cpu, and play a free-for-all of five against
// two classic Ace bots through RunMatch, the Postgres queue and the real
// engine. The match must finish with one result per slot and a sealed,
// published replay in which every neural bot plays.
func TestIntegration_NeuralBots_FiveShipMixedMatch(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	dbName := fmt.Sprintf("agentrix_neural_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()
	r.NoError(database.Migrate(conn.Db))
	seedSQL, err := os.ReadFile("../../script/data/00_seeds_postgresql.sql")
	r.NoError(err)
	_, err = conn.Db.Exec(string(seedSQL))
	r.NoError(err)

	r.NoError(game.GetRegistry().LoadGamesFromDir(resolveTestPath("games")))
	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	previousMaxTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = previousMaxTicks }()
	manifest.BinaryPath = resolveEngineBin(t)
	manifest.MaxTicks = 1800

	_, _, svc, queue := setupPostgresOrchestrationServer(t, conn, "worker-neural-01")
	defer queue.Close()

	// 1. Upload the three neural bundles as new versions of Ace agents.
	readFile := func(rel string) []byte {
		content, err := os.ReadFile(resolveTestPath(rel))
		r.NoError(err)
		return content
	}
	neural := "games/starfighter/examples/neural/"
	safetensorsZip := func() []byte {
		buf := new(bytes.Buffer)
		zw := zip.NewWriter(buf)
		for name, content := range map[string][]byte{
			"agentrix.json":            []byte(`{"name":"Neural Safetensors","entrypoint":"bot.py","protocol_version":"1.0","runtime":"python-ml-cpu"}`),
			"bot.py":                   readFile(neural + "bot_safetensors.py"),
			"policy.py":                readFile(neural + "policy.py"),
			"model/policy.safetensors": readFile(neural + "model/policy.safetensors"),
		} {
			f, err := zw.Create(name)
			r.NoError(err)
			_, err = f.Write(content)
			r.NoError(err)
		}
		r.NoError(zw.Close())
		return buf.Bytes()
	}()
	uploads := []struct {
		owner, agent string
		archive      []byte
	}{
		{"usr-pilot-001", "agent-star-ace-1", readFile("web/public/starfighter-neural-onnx.zip")},
		{"usr-pilot-002", "agent-star-ace-2", readFile("web/public/starfighter-neural-npz.zip")},
		{"usr-pilot-003", "agent-star-ace-3", safetensorsZip},
	}
	neuralSubs := make([]string, len(uploads))
	for i, up := range uploads {
		sub, err := svc.CreateSubmissionBundle(ctx, up.owner, "player", up.agent, up.archive)
		r.NoError(err, up.agent)
		r.Equal("validating", sub.Status)
		neuralSubs[i] = sub.Id
	}

	// 2. The worker admits them.
	for {
		processed, err := svc.ProcessNextAdmission(ctx)
		r.NoError(err)
		if !processed {
			break
		}
	}
	for _, id := range neuralSubs {
		sub, err := svc.GetSubmission(ctx, id)
		r.NoError(err)
		r.Equal("ready", sub.Status, "neural submission %s must be admitted: %s", id, sub.ErrorDetail)
	}

	// Classic Ace bots from the seeds, with an absolute code path.
	ace := resolveTestPath("games/starfighter/examples/bot_ace.py")
	_, err = conn.Db.Exec(`UPDATE submissions SET code_path = $1 WHERE id IN ('sub-star-ace-4', 'sub-star-ace-5')`, ace)
	r.NoError(err)

	// 3. Five slots, neural and classic interleaved.
	slots := []string{neuralSubs[0], "sub-star-ace-4", neuralSubs[1], "sub-star-ace-5", neuralSubs[2]}
	created, err := svc.CreateMatch(ctx, &model.Match{GameId: "starfighter", Seed: 2026}, slots)
	r.NoError(err)
	matchID := created.MatchId
	run, err := svc.RunMatch(ctx, matchID, "")
	r.NoError(err)
	job, err := queue.Dequeue(ctx)
	r.NoError(err)
	r.Equal(run.RunId, job.RunId)
	r.Equal(slots, job.SubmissionIds)
	r.NoError(executor.NewMatchExecutor(svc, executor.NewSandbox(0)).Execute(ctx, job))

	// 4. Finished match, one result per slot.
	match, err := svc.GetMatch(ctx, matchID)
	r.NoError(err)
	r.Equal("finished", match.Status)
	r.NotEmpty(match.ReplayId)

	rows, err := conn.Db.Query(`SELECT submission_id, "rank", score, status FROM results WHERE match_id = $1`, matchID)
	r.NoError(err)
	ranks := map[string]int{}
	for rows.Next() {
		var sub, status string
		var rank, score int
		r.NoError(rows.Scan(&sub, &rank, &score, &status))
		r.GreaterOrEqual(rank, 1)
		r.LessOrEqual(rank, 5)
		r.NotContains([]string{"crashed", "disqualified"}, status, "no bot may crash or be disqualified")
		ranks[sub] = rank
		t.Logf("result %s: rank=%d kills=%d status=%s", sub, rank, score, status)
	}
	r.NoError(rows.Close())
	r.Len(ranks, 5, "one result per slot")

	// 5. Sealed, published replay in which every neural bot plays.
	replay, err := svc.GetReplay(ctx, match.ReplayId)
	r.NoError(err)
	r.Len(replay.Sha256, 64)
	r.NotContains(replay.FilePath, "replays/tmp/", "the replay must be published, not left as a temporary file")
	raw, err := svc.StreamReplay(ctx, match.ReplayId)
	r.NoError(err)
	document, err := replaystream.DecodeNDJSON(bytes.NewReader(raw))
	r.NoError(err, "the mixed match must produce a sealed NDJSON replay")
	r.Len(document.Metadata.Participants, 5)
	r.Equal(document.Result.FinalStateHash, document.Snapshots[len(document.Snapshots)-1].StateHash)

	fired := map[int]int{}
	for _, snapshot := range document.Snapshots {
		var public struct {
			Events []struct {
				Type     string `json:"type"`
				PlayerID int    `json:"player_id"`
			} `json:"events"`
		}
		r.NoError(json.Unmarshal(snapshot.PublicSnapshot, &public))
		for _, event := range public.Events {
			if event.Type == "fired" {
				fired[event.PlayerID]++
			}
		}
	}
	for _, slot := range []int{0, 2, 4} {
		r.Greater(fired[slot], 0, "neural bot in slot %d (%s) must play and shoot", slot, slots[slot])
	}
	t.Logf("reason=%s ticks=%d fired=%v ranks=%v", document.Result.Reason, document.Result.FinalTick, fired, ranks)
}
