package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	replaystream "github.com/Agentrix-Group/Agentrix/src/replay"
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
	previousMaxTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = previousMaxTicks }()
	manifest.BinaryPath = engineBin
	// Limit max ticks to 5 for a fast integration test
	manifest.MaxTicks = 5

	replayPath := filepath.Join(t.TempDir(), "authoritative.ndjson")
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
		openReplayFn: func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			file, err := os.Create(replayPath)
			if err != nil {
				return nil, err
			}
			return replaystream.NewStreamWriter(file, metadata)
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
	replayFile, err := os.Open(replayPath)
	r.NoError(err)
	defer replayFile.Close()
	document, err := replaystream.DecodeNDJSON(replayFile)
	r.NoError(err, "the real match must produce a sealed NDJSON replay")
	r.Equal("starfighter", document.Metadata.GameID)
	r.Equal(17, document.Metadata.FixedTimestepMs)
	r.Equal(document.Result.FinalStateHash, document.Snapshots[len(document.Snapshots)-1].StateHash)

	// Verify that at least 3 valid ticks were simulated
	// Frame 0 is initial tick, frames 1..N are simulated ticks
	r.GreaterOrEqual(len(document.Snapshots), 4,
		"must have at least 4 replay frames (frame 0 initial + at least 3 simulated ticks)")

	t.Logf("Total simulated frames: %d", len(document.Snapshots))

	// Verify frames 1, 2, 3 have valid action records without tick desync
	for tick := 1; tick <= 3; tick++ {
		frame := document.Snapshots[tick]
		r.Equal(tick, frame.Tick, "frame tick must match sequentially")
		r.NotEmpty(frame.PublicSnapshot)
		r.NotEmpty(frame.StateHash)
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
	previousMaxTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = previousMaxTicks }()
	manifest.BinaryPath = engineBin
	manifest.MaxTicks = 4

	capturedReplay := &captureReplayWriter{}
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
		openReplayFn: func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			return capturedReplay, nil
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
	r.True(capturedReplay.completed)
	r.GreaterOrEqual(len(capturedReplay.snapshots), 4, "must have at least 4 replay frames")

	// Both fallback bots (bot-ref-1 and bot-ref-2) must have valid actions without error
	for tick := 1; tick <= 3; tick++ {
		frame := capturedReplay.snapshots[tick]
		r.Equal(tick, frame.Tick)
		r.NotEmpty(frame.PublicSnapshot)
	}
}

func TestStarfighterIntegration_TimeoutDisqualifiesAndSealsReplay(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	engineBin := resolveTestPath("bin/starfighter-engine")
	if _, err := os.Stat(engineBin); err != nil {
		t.Skipf("Starfighter engine binary not found at %s. Run 'make build-engine' first.", engineBin)
	}
	gamesDir := resolveTestPath("games")
	r.NoError(game.GetRegistry().LoadGamesFromDir(gamesDir))
	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	previousMaxTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = previousMaxTicks }()
	manifest.MaxTicks = 50

	slowBot := filepath.Join(t.TempDir(), "slow_bot.py")
	r.NoError(os.WriteFile(slowBot, []byte(`import json, sys, time
json.loads(sys.stdin.readline())
for line in sys.stdin:
    message = json.loads(line)
    if message["type"] == "end":
        break
    time.sleep(2)
`), 0o600))
	hunter := resolveTestPath("games/starfighter/examples/bot_hunter.py")
	replayPath := filepath.Join(t.TempDir(), "timeout.ndjson")

	mockSvc := &mockExecutorService{
		getMatchFn: func(context.Context, string) (*model.Match, error) {
			return &model.Match{Id: "match-timeout", GameId: "starfighter", Status: common.MatchStatusPending}, nil
		},
		getSubmissionFn: func(_ context.Context, id string) (*model.Submission, error) {
			path := hunter
			if id == "slow" {
				path = slowBot
			}
			return &model.Submission{Id: id, AgentId: id, Language: "python", CodePath: path, Status: common.SubmissionStatusReady, Active: true}, nil
		},
		openReplayFn: func(_ context.Context, _ *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			file, err := os.Create(replayPath)
			if err != nil {
				return nil, err
			}
			return replaystream.NewStreamWriter(file, metadata)
		},
	}
	engineFactory := func(ctx context.Context, _ *connection.MatchJob) (engine.EngineClient, error) {
		client := engine.NewSubprocessClient()
		err := client.Start(ctx, engine.StartConfig{BinaryPath: engineBin, HandshakeTimeout: 5 * time.Second})
		return client, err
	}
	exec := NewMatchExecutor(mockSvc, NewSandbox(500*time.Millisecond), engineFactory)
	started := time.Now()
	err := exec.Execute(ctx, &connection.MatchJob{
		JobId: "job-timeout", MatchId: "match-timeout", GameId: "starfighter",
		SubmissionIds: []string{"slow", "hunter"}, Seed: 41,
	})
	r.NoError(err)
	r.Less(time.Since(started), 2*time.Second, "the slow bot must be killed before its sleep completes")

	file, err := os.Open(replayPath)
	r.NoError(err)
	defer file.Close()
	document, err := replaystream.DecodeNDJSON(file)
	r.NoError(err)
	r.Equal("timeout", document.Result.Reason)
	r.Equal("hunter", document.Result.Winner)
	r.Equal(1, document.Result.FinalTick)
}

func TestStarfighterIntegration_DoubleTimeoutDisqualifiesBothWithNoWinner(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	engineBin := resolveTestPath("bin/starfighter-engine")
	if _, err := os.Stat(engineBin); err != nil {
		t.Skipf("Starfighter engine binary not found at %s. Run 'make build-engine' first.", engineBin)
	}
	gamesDir := resolveTestPath("games")
	r.NoError(game.GetRegistry().LoadGamesFromDir(gamesDir))
	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	previousMaxTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = previousMaxTicks }()
	manifest.MaxTicks = 50

	slowBot := filepath.Join(t.TempDir(), "slow_bot_all.py")
	r.NoError(os.WriteFile(slowBot, []byte(`import json, sys, time
json.loads(sys.stdin.readline())
for line in sys.stdin:
    message = json.loads(line)
    if message["type"] == "end":
        break
    time.sleep(2)
`), 0o600))
	replayPath := filepath.Join(t.TempDir(), "double_timeout.ndjson")
	var savedResults []*model.Result

	mockSvc := &mockExecutorService{
		getMatchFn: func(context.Context, string) (*model.Match, error) {
			return &model.Match{Id: "match-double-timeout", GameId: "starfighter", Status: common.MatchStatusPending}, nil
		},
		getSubmissionFn: func(_ context.Context, id string) (*model.Submission, error) {
			return &model.Submission{Id: id, AgentId: id, Language: "python", CodePath: slowBot, Status: common.SubmissionStatusReady, Active: true}, nil
		},
		openReplayFn: func(_ context.Context, _ *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			file, err := os.Create(replayPath)
			if err != nil {
				return nil, err
			}
			return replaystream.NewStreamWriter(file, metadata)
		},
		createResultFn: func(_ context.Context, res *model.Result) error {
			savedResults = append(savedResults, res)
			return nil
		},
	}

	exec := NewMatchExecutor(mockSvc, NewSandbox(300*time.Millisecond), func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		client := engine.NewSubprocessClient()
		err := client.Start(ctx, engine.StartConfig{BinaryPath: engineBin, HandshakeTimeout: 5 * time.Second})
		return client, err
	})

	job := &connection.MatchJob{
		JobId:         "job-double-timeout",
		MatchId:       "match-double-timeout",
		GameId:        "starfighter",
		SubmissionIds: []string{"slow-1", "slow-2"},
		Seed:          123,
	}

	r.NoError(exec.Execute(ctx, job))

	replayFile, err := os.Open(replayPath)
	r.NoError(err)
	defer replayFile.Close()
	document, err := replaystream.DecodeNDJSON(replayFile)
	r.NoError(err)
	r.Equal("timeout", document.Result.Reason)
	r.Empty(document.Result.Winner, "no winner should be declared when both bots timeout simultaneously")
	r.Equal(1, document.Result.FinalTick)

	// Verify both results were saved with tied rank 1 and score 0
	r.Equal(2, len(savedResults))
	for _, res := range savedResults {
		r.Equal(1, res.Rank, "both players must be tied at rank 1")
		r.Equal(0, res.Score, "score must be 0 for disqualified bots")
	}
}


func TestStarfighterIntegration_50ContinuousTicksBetweenHunterAndEvasive(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	engineBin := resolveTestPath("bin/starfighter-engine")
	if _, err := os.Stat(engineBin); err != nil {
		t.Skipf("Starfighter engine binary not found at %s. Run 'make build-engine' first.", engineBin)
	}

	botHunter := resolveTestPath("games/starfighter/examples/bot_hunter.py")
	botEvasive := resolveTestPath("games/starfighter/examples/bot_evasive.py")
	r.FileExists(botHunter, "bot_hunter.py must exist")
	r.FileExists(botEvasive, "bot_evasive.py must exist")

	gamesDir := resolveTestPath("games")
	_ = game.GetRegistry().LoadGamesFromDir(gamesDir)

	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest, "starfighter manifest must be loaded")
	previousMaxTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = previousMaxTicks }()
	manifest.BinaryPath = engineBin
	// Configure exactly 50 continuous ticks as mandated by Phase 1
	manifest.MaxTicks = 50

	replayPath := filepath.Join(t.TempDir(), "50_ticks_authoritative.ndjson")
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
			if id == "sub-evasive" {
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
		openReplayFn: func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			file, err := os.Create(replayPath)
			if err != nil {
				return nil, err
			}
			replay.FilePath = replayPath
			return replaystream.NewStreamWriter(file, metadata)
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
		JobId:         "job-starfighter-50-ticks",
		Attempt:       1,
		MatchId:       "match-starfighter-50-ticks",
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-hunter", "sub-evasive"},
		Seed:          2026,
	}

	start := time.Now()
	err := exec.Execute(ctx, job)
	r.NoError(err, "match execution for 50 continuous ticks must succeed without error")
	elapsed := time.Since(start)
	t.Logf("Simulated 50 continuous ticks in %v", elapsed)

	// Status and persistence checks
	r.Contains(updatedStatuses, common.MatchStatusRunning)
	r.Contains(updatedStatuses, common.MatchStatusFinished)
	r.Equal(2, resultsCreated, "both player results must be created")

	replayFile, err := os.Open(replayPath)
	r.NoError(err)
	defer replayFile.Close()
	document, err := replaystream.DecodeNDJSON(replayFile)
	r.NoError(err, "50-tick match must produce a valid sealed NDJSON replay")
	r.Equal("starfighter", document.Metadata.GameID)
	r.Equal(17, document.Metadata.FixedTimestepMs)
	r.NotNil(document.Result, "result footer must be present")
	r.Equal(document.Result.FinalStateHash, document.Snapshots[len(document.Snapshots)-1].StateHash)

	// Expect 51 snapshots (tick 0 through tick 50)
	r.Equal(51, len(document.Snapshots), "must have exactly 51 replay snapshots (tick 0 to 50)")

	// Strictly verify sequential tick ordering and non-empty state hashes and snapshots
	for expectedTick := 0; expectedTick <= 50; expectedTick++ {
		frame := document.Snapshots[expectedTick]
		r.Equal(expectedTick, frame.Tick, "tick index must match sequentially without gaps or desync")
		r.NotEmpty(frame.StateHash, "state hash must be populated for tick %d", expectedTick)
		r.NotEmpty(frame.PublicSnapshot, "public snapshot must be present for tick %d", expectedTick)
	}

	r.Equal(50, document.Result.FinalTick, "final tick must be 50")

	// Verify Zstandard compression was performed upon completion
	if replaystream.IsZstdAvailable() {
		zstPath := replayPath + ".zst"
		r.FileExists(zstPath, "authoritative NDJSON replay must be compressed to .zst upon match completion")
		decompressed, decErr := replaystream.DecompressZstd(ctx, zstPath)
		r.NoError(decErr, "decompression of replay .zst must succeed")
		r.NotEmpty(decompressed, "decompressed replay payload must not be empty")
	}
}

func TestStarfighterIntegration_100MatchesDurationAndMemoryMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 100 matches metrics test in short mode")
	}
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	engineBin := resolveTestPath("bin/starfighter-engine")
	if _, err := os.Stat(engineBin); err != nil {
		t.Skipf("Starfighter engine binary not found at %s. Run 'make build-engine' first.", engineBin)
	}

	botHunter := resolveTestPath("games/starfighter/examples/bot_hunter.py")
	botEvasive := resolveTestPath("games/starfighter/examples/bot_evasive.py")

	gamesDir := resolveTestPath("games")
	_ = game.GetRegistry().LoadGamesFromDir(gamesDir)

	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	prevTicks := manifest.MaxTicks
	defer func() { manifest.MaxTicks = prevTicks }()
	manifest.BinaryPath = engineBin
	manifest.MaxTicks = 5

	const matchCount = 100
	durations := make([]time.Duration, matchCount)
	var peakAlloc uint64
	var peakSys uint64

	sandbox := NewSandbox(2 * time.Second)
	engineFactory := func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		client := engine.NewSubprocessClient()
		err := client.Start(ctx, engine.StartConfig{
			BinaryPath:       engineBin,
			HandshakeTimeout: 5 * time.Second,
		})
		return client, err
	}

	tempDir := t.TempDir()

	for i := 0; i < matchCount; i++ {
		matchID := fmt.Sprintf("match-baseline-%03d", i)
		replayPath := filepath.Join(tempDir, matchID+".ndjson")
		mockSvc := &mockExecutorService{
			getMatchFn: func(ctx context.Context, id string) (*model.Match, error) {
				return &model.Match{Id: id, GameId: "starfighter", Status: common.MatchStatusPending}, nil
			},
			updateMatchFn: func(ctx context.Context, match *model.Match) error {
				return nil
			},
			getSubmissionFn: func(ctx context.Context, id string) (*model.Submission, error) {
				codePath := botHunter
				if id == "sub-2" {
					codePath = botEvasive
				}
				return &model.Submission{
					Id: id, AgentId: id + "-agent", Language: "python", CodePath: codePath, Status: common.SubmissionStatusReady, Active: true,
				}, nil
			},
			openReplayFn: func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
				file, err := os.Create(replayPath)
				if err != nil {
					return nil, err
				}
				replay.FilePath = replayPath
				return replaystream.NewStreamWriter(file, metadata)
			},
			createResultFn: func(ctx context.Context, res *model.Result) error {
				return nil
			},
		}

		exec := NewMatchExecutor(mockSvc, sandbox, engineFactory)
		job := &connection.MatchJob{
			JobId:         fmt.Sprintf("job-baseline-%03d", i),
			Attempt:       1,
			MatchId:       matchID,
			GameId:        "starfighter",
			SubmissionIds: []string{"sub-1", "sub-2"},
			Seed:          int64(1000 + i),
		}

		start := time.Now()
		err := exec.Execute(ctx, job)
		durations[i] = time.Since(start)
		r.NoError(err, "match %d execution must succeed", i)

		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		if m.Alloc > peakAlloc {
			peakAlloc = m.Alloc
		}
		if m.Sys > peakSys {
			peakSys = m.Sys
		}
		_ = os.Remove(replayPath)
	}

	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	p50 := durations[matchCount*50/100]
	p95 := durations[matchCount*95/100]
	p99 := durations[matchCount*99/100]
	minDur := durations[0]
	maxDur := durations[matchCount-1]

	t.Logf("=== 100 MATCHES BASELINE METRICS ===")
	t.Logf("Matches: %d", matchCount)
	t.Logf("Min duration: %v", minDur)
	t.Logf("p50 duration: %v", p50)
	t.Logf("p95 duration: %v", p95)
	t.Logf("p99 duration: %v", p99)
	t.Logf("Max duration: %v", maxDur)
	t.Logf("Peak Alloc: %.2f MB", float64(peakAlloc)/(1024*1024))
	t.Logf("Peak Sys: %.2f MB", float64(peakSys)/(1024*1024))
}
