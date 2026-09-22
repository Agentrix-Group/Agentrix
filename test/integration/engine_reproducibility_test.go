package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func resolveTestPath(rel string) string {
	if _, err := os.Stat(rel); err == nil {
		return rel
	}
	if _, err := os.Stat(filepath.Join("../..", rel)); err == nil {
		return filepath.Join("../..", rel)
	}
	return rel
}

func resolveEngineBin(t *testing.T) string {
	candidates := []string{
		"bin/starfighter-engine",
		"../../bin/starfighter-engine",
		"../agentrix_engine/target/debug/starfighter-engine",
		"../../agentrix_engine/target/debug/starfighter-engine",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	t.Skip("Starfighter engine binary not found for integration test")
	return ""
}

func TestIntegration_Engine_ReproducibleExecutionWithSpec(t *testing.T) {
	r := require.New(t)
	dbName := fmt.Sprintf("agentrix_engine_spec_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	engineBin := resolveEngineBin(t)
	t.Setenv("AGENTRIX_ENGINE_BIN", engineBin)
	engineDigest, err := model.ComputeFileSHA256(engineBin)
	r.NoError(err)
	r.NotEmpty(engineDigest)

	// Ensure game manifest is loaded and points to engine binary
	manifestsDir := resolveTestPath("games")
	r.NoError(game.GetRegistry().LoadGamesFromDir(manifestsDir))
	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	manifest.BinaryPath = engineBin
	previousMaxTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = previousMaxTicks }()
	manifest.MaxTicks = 10 // Fast integration simulation

	// Setup server and worker
	_, _, svc, queue := setupPostgresOrchestrationServer(t, conn, "worker-engine-spec-01")
	defer queue.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// 1. Create match with 2 participants
	match := &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
		Seed:      42,
	}
	createResp, err := svc.CreateMatch(ctx, match, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	r.NotEmpty(createResp.MatchId)
	matchID := createResp.MatchId

	// 2. Initiate execution via RunMatch
	resp, err := svc.RunMatch(ctx, matchID, "idem-spec-key-1")
	r.NoError(err)
	r.Equal("queued", resp.Status)
	r.NotEmpty(resp.RunId)
	r.NotEmpty(resp.JobId)

	// 3. Verify ExecutionSpec in database
	run, err := svc.GetMatchRun(ctx, resp.RunId)
	r.NoError(err)
	r.NotNil(run)
	r.NotEmpty(run.ExecutionSpec, "match_runs.execution_spec must be populated")

	// Parse and validate spec
	spec, err := model.ParseExecutionSpec(run.ExecutionSpec)
	r.NoError(err, "ExecutionSpec must be valid JSON and validate cleanly")
	r.Equal(model.ExecutionSpecProtocolVersion, spec.ProtocolVersion)
	r.Equal("starfighter", spec.Game.GameID)
	r.Equal(uint64(42), spec.Seed)
	r.Equal(uint64(manifest.MaxTicks), spec.Limits.MaxTicks)
	r.Equal(engineDigest, spec.EngineDigest)
	r.NotEmpty(spec.ConfigDigest)
	specDigest, err := spec.ComputeDigest()
	r.NoError(err)
	r.NotEmpty(specDigest)
	r.Len(spec.Slots, 2)
	r.Equal("sub-alice-01", spec.Slots[0].SlotID)
	r.Equal("sub-bob-01", spec.Slots[1].SlotID)

	// 4. Worker reserves and executes the job with real Rust engine
	job, err := queue.Dequeue(ctx)
	r.NoError(err)
	r.NotNil(job)
	r.Equal(matchID, job.MatchId)
	r.Equal(engineDigest, job.EngineDigest)

	matchExec := executor.NewMatchExecutor(svc, executor.NewSandbox(0))
	r.NoError(matchExec.Execute(ctx, job))

	// 5. Verify database commit invariants
	committedRun, err := svc.GetMatchRun(ctx, resp.RunId)
	r.NoError(err)
	r.Equal(model.MatchRunStatusCommitted, committedRun.Status)

	dbMatch, err := svc.GetMatch(ctx, matchID)
	r.NoError(err)
	r.Equal(common.MatchStatusFinished, dbMatch.Status)
	r.NotNil(dbMatch.CommittedRunId)
	r.Equal(resp.RunId, *dbMatch.CommittedRunId)
	r.NotEmpty(dbMatch.ReplayId)

	// 6. Verify Results table references match_run_id and slot_id
	var resultCount int
	var runMatchedCount int
	var slotMatchedCount int
	err = conn.Db.QueryRow(`
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE match_run_id = $1),
		       COUNT(*) FILTER (WHERE slot_id IS NOT NULL)
		FROM results
		WHERE match_id = $2
	`, resp.RunId, matchID).Scan(&resultCount, &runMatchedCount, &slotMatchedCount)
	r.NoError(err)
	r.Equal(2, resultCount)
	r.Equal(2, runMatchedCount)
	r.Equal(2, slotMatchedCount)

	// 7. Verify Replay metadata contains RunID, EngineDigest, ConfigHash
	replayDoc, err := svc.GetReplay(ctx, dbMatch.ReplayId)
	r.NoError(err)
	r.NotNil(replayDoc)
	r.Equal(matchID, replayDoc.MatchId)
	r.NotEmpty(replayDoc.Sha256)
	r.Greater(replayDoc.DurationTicks, 0)
}

func TestIntegration_Engine_DigestMismatchExplicitFailure(t *testing.T) {
	r := require.New(t)
	dbName := fmt.Sprintf("agentrix_engine_mismatch_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	engineBin := resolveEngineBin(t)
	manifestsDir := resolveTestPath("games")
	r.NoError(game.GetRegistry().LoadGamesFromDir(manifestsDir))
	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	manifest.BinaryPath = engineBin

	_, _, svc, queue := setupPostgresOrchestrationServer(t, conn, "worker-mismatch-01")
	defer queue.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	createResp, err := svc.CreateMatch(ctx, &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
		Seed:      99,
	}, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	matchID := createResp.MatchId

	resp, err := svc.RunMatch(ctx, matchID, "")
	r.NoError(err)

	// Corrupt engine digest in queued job
	job, err := queue.Dequeue(ctx)
	r.NoError(err)
	job.EngineDigest = "bad0000000000000000000000000000000000000000000000000000000000000"

	matchExec := executor.NewMatchExecutor(svc, executor.NewSandbox(0))
	execErr := matchExec.Execute(ctx, job)
	r.Error(execErr)
	r.ErrorIs(execErr, engine.ErrEngineDigestMismatch)

	// Verify match_run status is failed
	failedRun, err := svc.GetMatchRun(ctx, resp.RunId)
	r.NoError(err)
	r.Equal(model.MatchRunStatusFailed, failedRun.Status)
	r.Contains(failedRun.LastError, "engine binary digest mismatch")

	// Verify match status is failed
	dbMatch, err := svc.GetMatch(ctx, matchID)
	r.NoError(err)
	r.Equal(common.MatchStatusFailed, dbMatch.Status)
	r.Nil(dbMatch.CommittedRunId)

	// Zero results committed
	var resultCount int
	r.NoError(conn.Db.QueryRow("SELECT COUNT(*) FROM results WHERE match_id = $1", matchID).Scan(&resultCount))
	r.Equal(0, resultCount)
}

func TestIntegration_Engine_IncompatibleProtocolVersionRejected(t *testing.T) {
	r := require.New(t)
	dbName := fmt.Sprintf("agentrix_engine_proto_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	// Create shell script that outputs an incompatible protocol engine_ready
	incompatibleEngineScript := filepath.Join(t.TempDir(), "incompatible-engine.sh")
	scriptContent := `#!/bin/sh
printf '{"protocolVersion":"agentrix-engine/99.0","type":"engine_ready","matchId":"","sequence":1,"payload":{"engineVersion":"99.0.0","supportedProtocols":["agentrix-engine/99.0"]}}\n'
while read line; do :; done
`
	r.NoError(os.WriteFile(incompatibleEngineScript, []byte(scriptContent), 0755))

	_, _, svc, queue := setupPostgresOrchestrationServer(t, conn, "worker-proto-01")
	defer queue.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	createResp, err := svc.CreateMatch(ctx, &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
	}, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	matchID := createResp.MatchId

	resp, err := svc.RunMatch(ctx, matchID, "")
	r.NoError(err)

	job, err := queue.Dequeue(ctx)
	r.NoError(err)

	// Custom engine factory using the incompatible script
	customFactory := func(ctx context.Context, j *connection.MatchJob) (engine.EngineClient, error) {
		client := engine.NewSubprocessClient()
		startErr := client.Start(ctx, engine.StartConfig{
			BinaryPath:       incompatibleEngineScript,
			HandshakeTimeout: 3 * time.Second,
		})
		if startErr != nil {
			return nil, startErr
		}
		return client, nil
	}

	matchExec := executor.NewMatchExecutor(svc, executor.NewSandbox(0), customFactory)
	execErr := matchExec.Execute(ctx, job)
	r.Error(execErr)
	r.ErrorIs(execErr, engine.ErrIncompatibleVersion)

	// Verify match_run status is failed
	failedRun, err := svc.GetMatchRun(ctx, resp.RunId)
	r.NoError(err)
	r.Equal(model.MatchRunStatusFailed, failedRun.Status)

	dbMatch, err := svc.GetMatch(ctx, matchID)
	r.NoError(err)
	r.Equal(common.MatchStatusFailed, dbMatch.Status)
	r.Nil(dbMatch.CommittedRunId)
}

func TestIntegration_Engine_AtomicReplayRollbackOnExecutionFailure(t *testing.T) {
	r := require.New(t)
	dbName := fmt.Sprintf("agentrix_engine_abort_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	seedOrchestrationBaseData(t, conn)

	engineBin := resolveEngineBin(t)
	manifestsDir := resolveTestPath("games")
	r.NoError(game.GetRegistry().LoadGamesFromDir(manifestsDir))
	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	manifest.BinaryPath = engineBin

	_, _, svc, queue := setupPostgresOrchestrationServer(t, conn, "worker-abort-01")
	defer queue.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	createResp, err := svc.CreateMatch(ctx, &model.Match{
		ContestId: "contest-orch-01",
		GameId:    "starfighter",
	}, []string{"sub-alice-01", "sub-bob-01"})
	r.NoError(err)
	matchID := createResp.MatchId

	resp, err := svc.RunMatch(ctx, matchID, "")
	r.NoError(err)

	job, err := queue.Dequeue(ctx)
	r.NoError(err)

	failingEngine := &testFailingAdvanceTickEngine{}
	customFactory := func(ctx context.Context, j *connection.MatchJob) (engine.EngineClient, error) {
		return failingEngine, nil
	}

	matchExec := executor.NewMatchExecutor(svc, executor.NewSandbox(0), customFactory)
	execErr := matchExec.Execute(ctx, job)
	r.Error(execErr)
	r.Contains(execErr.Error(), "simulation crashed on advance tick")

	// Verify no replay was published in database
	dbMatch, err := svc.GetMatch(ctx, matchID)
	r.NoError(err)
	r.Equal(common.MatchStatusFailed, dbMatch.Status)
	r.Empty(dbMatch.ReplayId)

	// Verify no replay records in replays table for this match
	var replayCount int
	r.NoError(conn.Db.QueryRow("SELECT COUNT(*) FROM replays WHERE match_id = $1", matchID).Scan(&replayCount))
	r.Equal(0, replayCount)

	// Verify no committed run
	r.Nil(dbMatch.CommittedRunId)
	failedRun, err := svc.GetMatchRun(ctx, resp.RunId)
	r.NoError(err)
	r.Equal(model.MatchRunStatusFailed, failedRun.Status)
}

type testFailingAdvanceTickEngine struct{}

func (e *testFailingAdvanceTickEngine) Start(ctx context.Context, cfg engine.StartConfig) error {
	return nil
}
func (e *testFailingAdvanceTickEngine) InitializeMatch(ctx context.Context, req engine.InitializeMatchRequest) (*engine.MatchInitializedResult, error) {
	return &engine.MatchInitializedResult{
		MatchID:        req.MatchID,
		InitialTick:    0,
		StateHash:      "hash-0",
		PublicSnapshot: []byte(`{"tick":0,"stateHash":"hash-0"}`),
		Perceptions: map[string]json.RawMessage{
			"sub-alice-01": []byte(`{"tick":0}`),
			"sub-bob-01":   []byte(`{"tick":0}`),
		},
	}, nil
}
func (e *testFailingAdvanceTickEngine) AdvanceTick(ctx context.Context, req engine.AdvanceTickRequest) (*engine.TickResult, error) {
	return nil, fmt.Errorf("simulation crashed on advance tick")
}
func (e *testFailingAdvanceTickEngine) FinishMatch(ctx context.Context, reason string) (*engine.MatchResult, error) {
	return nil, fmt.Errorf("engine closed")
}
func (e *testFailingAdvanceTickEngine) Close(ctx context.Context) error {
	return nil
}
func (e *testFailingAdvanceTickEngine) EngineVersion() string {
	return "0.3.0"
}
func (e *testFailingAdvanceTickEngine) EngineDigest() string {
	return "mock-digest"
}
