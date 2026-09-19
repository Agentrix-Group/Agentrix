package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/model"
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

func TestStarfighterIntegration_RealRustEngineAndPythonBots(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	engineBin := resolveTestPath("bin/starfighter-engine")
	if _, err := os.Stat(engineBin); err != nil {
		t.Skipf("Starfighter engine binary not found at %s. Run 'make build-engine' first.", engineBin)
	}

	botHunter := resolveTestPath("games/starfighter/examples/bot_hunter.py")
	botEvasive := resolveTestPath("games/starfighter/examples/bot_evasive.py")
	r.FileExists(botHunter, "bot_hunter.py must exist")
	r.FileExists(botEvasive, "bot_evasive.py must exist")

	// Load game manifests
	gamesDir := resolveTestPath("games")
	_ = game.GetRegistry().LoadGamesFromDir(gamesDir)

	// Ensure manifest for starfighter has the correct resolved binary path
	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest, "starfighter manifest must be loaded")
	manifest.BinaryPath = engineBin
	// Limit max ticks to 5 for a fast integration test
	manifest.MaxTicks = 5

	var capturedReplay *model.ReplayData
	resultsCreated := 0
	updatedStatuses := make([]string, 0)

	mockSvc := &mockExecutorService{
		getMatchFn: func(ctx context.Context, id string) (*model.Match, error) {
			return &model.Match{
				Id:     id,
				GameId: "starfighter",
				Status: common.MatchStatusPending,
			}, nil
		},
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
		},
		getSubmissionFn: func(ctx context.Context, id string) (*model.Submission, error) {
			codePath := botHunter
			if id == "sub-star-2" {
				codePath = botEvasive
			}
			return &model.Submission{
				Id:       id,
				AgentId:  id + "-agent",
				Language: "python",
				CodePath: codePath,
				Status:   common.SubmissionStatusReady,
				Active:   true,
			}, nil
		},
		saveReplayFn: func(ctx context.Context, replay *model.Replay, data *model.ReplayData) error {
			capturedReplay = data
			return nil
		},
		createResultFn: func(ctx context.Context, res *model.Result) error {
			resultsCreated++
			return nil
		},
	}

	sandbox := NewSandbox(3 * time.Second)

	engineFactory := func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		client := engine.NewSubprocessClient()
		err := client.Start(ctx, engine.StartConfig{
			BinaryPath:       engineBin,
			HandshakeTimeout: 5 * time.Second,
		})
		return client, err
	}

	exec := NewMatchExecutor(mockSvc, sandbox, engineFactory)
	r.NotNil(exec)

	job := &connection.MatchJob{
		JobId:         "job-starfighter-e2e",
		Attempt:       1,
		MatchId:       "match-starfighter-e2e",
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-star-1", "sub-star-2"},
		Seed:          1337,
	}

	err := exec.Execute(ctx, job)
	r.NoError(err, "match execution must succeed without errors")

	// Verification
	r.Contains(updatedStatuses, common.MatchStatusRunning)
	r.Contains(updatedStatuses, common.MatchStatusFinished)
	r.Equal(2, resultsCreated, "both player results must be created")
	r.NotNil(capturedReplay, "replay data must be saved")

	// Verify that at least 3 valid ticks were simulated
	// Frame 0 is initial tick, frames 1..N are simulated ticks
	r.GreaterOrEqual(len(capturedReplay.Frames), 4,
		"must have at least 4 replay frames (frame 0 initial + at least 3 simulated ticks)")

	t.Logf("Total simulated frames: %d", len(capturedReplay.Frames))

	// Verify frames 1, 2, 3 have valid action records without tick desync
	for tick := 1; tick <= 3; tick++ {
		frame := capturedReplay.Frames[tick]
		r.Equal(tick, frame.Tick, "frame tick must match sequentially")
		r.NotEmpty(frame.Actions, "actions must be recorded for tick %d", tick)
		r.NotNil(frame.Actions["sub-star-1"], "sub-star-1 must have action")
		r.NotNil(frame.Actions["sub-star-2"], "sub-star-2 must have action")
		// Verify neither was marked as REST due to an invalid action / tick mismatch
		r.NotEqual("REST", frame.Actions["sub-star-1"], "sub-star-1 must have valid non-REST action on tick %d", tick)
		r.NotEqual("REST", frame.Actions["sub-star-2"], "sub-star-2 must have valid non-REST action on tick %d", tick)
	}
}

func TestStarfighterIntegration_WithFallbackBots(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	engineBin := resolveTestPath("bin/starfighter-engine")
	if _, err := os.Stat(engineBin); err != nil {
		t.Skipf("Starfighter engine binary not found at %s. Run 'make build-engine' first.", engineBin)
	}

	// Load game manifests
	gamesDir := resolveTestPath("games")
	_ = game.GetRegistry().LoadGamesFromDir(gamesDir)

	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	manifest.BinaryPath = engineBin
	manifest.MaxTicks = 4

	var capturedReplay *model.ReplayData
	mockSvc := &mockExecutorService{
		getMatchFn: func(ctx context.Context, id string) (*model.Match, error) {
			return &model.Match{
				Id:     id,
				GameId: "starfighter",
				Status: common.MatchStatusPending,
			}, nil
		},
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			return nil
		},
		saveReplayFn: func(ctx context.Context, replay *model.Replay, data *model.ReplayData) error {
			capturedReplay = data
			return nil
		},
		createResultFn: func(ctx context.Context, res *model.Result) error {
			return nil
		},
	}

	sandbox := NewSandbox(3 * time.Second)
	engineFactory := func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		client := engine.NewSubprocessClient()
		err := client.Start(ctx, engine.StartConfig{
			BinaryPath:       engineBin,
			HandshakeTimeout: 5 * time.Second,
		})
		return client, err
	}

	exec := NewMatchExecutor(mockSvc, sandbox, engineFactory)

	// No submissions provided: executor must load fallback reference bots compatible with Starfighter!
	job := &connection.MatchJob{
		JobId:         "job-starfighter-fallback",
		Attempt:       1,
		MatchId:       "match-starfighter-fallback",
		GameId:        "starfighter",
		SubmissionIds: []string{},
		Seed:          999,
	}

	err := exec.Execute(ctx, job)
	r.NoError(err, "match execution with fallback bots must succeed")
	r.NotNil(capturedReplay)
	r.GreaterOrEqual(len(capturedReplay.Frames), 4, "must have at least 4 replay frames")

	// Both fallback bots (bot-ref-1 and bot-ref-2) must have valid actions without error
	for tick := 1; tick <= 3; tick++ {
		frame := capturedReplay.Frames[tick]
		r.NotNil(frame.Actions["bot-ref-1"])
		r.NotNil(frame.Actions["bot-ref-2"])
		r.NotEqual("REST", frame.Actions["bot-ref-1"])
		r.NotEqual("REST", frame.Actions["bot-ref-2"])
	}
}
