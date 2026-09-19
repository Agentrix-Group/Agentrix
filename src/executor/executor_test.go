package executor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/model"
	replaystream "github.com/F4nk1/Agentrix/src/replay"
	"github.com/F4nk1/Agentrix/src/repository"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

func init() {
	game.GetRegistry().RegisterManifest(&game.Manifest{
		ID: "starfighter", MinPlayers: 2, MaxPlayers: 2, MaxTicks: 2, FixedTimestepMs: 17,
		ReferenceAgents: []game.ReferenceAgent{
			{ID: "hunter", Path: "games/starfighter/examples/bot_hunter.py"},
			{ID: "evasive", Path: "games/starfighter/examples/bot_evasive.py"},
		},
	})
}

type mockExecutorService struct {
	service.Service
	getMatchFn          func(ctx context.Context, id string) (*model.Match, error)
	updateMatchFn       func(ctx context.Context, match *model.Match) error
	getSubmissionFn     func(ctx context.Context, id string) (*model.Submission, error)
	openReplayFn        func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error)
	createResultFn      func(ctx context.Context, res *model.Result) error
	calculateRankingsFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
	commitMatchResultFn func(ctx context.Context, commit model.MatchResultCommit) error
	publishReplayFn     func(ctx context.Context, replayID string) (*model.Replay, error)
	discardReplayFn     func(ctx context.Context, replayID string) error
}

func (m *mockExecutorService) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	if m.getMatchFn != nil {
		return m.getMatchFn(ctx, id)
	}
	return &model.Match{Id: id, Status: common.MatchStatusPending}, nil
}

func (m *mockExecutorService) UpdateMatch(ctx context.Context, match *model.Match) error {
	if m.updateMatchFn != nil {
		return m.updateMatchFn(ctx, match)
	}
	return nil
}

func (m *mockExecutorService) CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error {
	if m.commitMatchResultFn != nil {
		return m.commitMatchResultFn(ctx, commit)
	}
	if m.updateMatchFn != nil {
		_ = m.updateMatchFn(ctx, &model.Match{Id: commit.MatchID, Status: commit.Status, ReplayId: commit.ReplayID, FinishedAt: &commit.FinishedAt})
	}
	if m.createResultFn != nil {
		for _, res := range commit.Results {
			rCopy := res
			_ = m.createResultFn(ctx, &rCopy)
		}
	}
	return nil
}

func (m *mockExecutorService) PublishReplay(ctx context.Context, replayID string) (*model.Replay, error) {
	if m.publishReplayFn != nil {
		return m.publishReplayFn(ctx, replayID)
	}
	return &model.Replay{Id: replayID}, nil
}

func (m *mockExecutorService) DiscardReplay(ctx context.Context, replayID string) error {
	if m.discardReplayFn != nil {
		return m.discardReplayFn(ctx, replayID)
	}
	return nil
}

func (m *mockExecutorService) CreateMatchRun(ctx context.Context, run *model.MatchRun) error {
	return nil
}

func (m *mockExecutorService) GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error) {
	return nil, nil
}

func (m *mockExecutorService) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	if m.getSubmissionFn != nil {
		return m.getSubmissionFn(ctx, id)
	}
	return &model.Submission{Id: id, CodePath: "fake.py"}, nil
}

func (m *mockExecutorService) OpenReplay(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
	if m.openReplayFn != nil {
		return m.openReplayFn(ctx, replay, metadata)
	}
	return &captureReplayWriter{}, nil
}

type captureReplayWriter struct {
	snapshots []model.ReplaySnapshot
	result    model.ReplayResult
	completed bool
}

func (w *captureReplayWriter) WriteSnapshot(snapshot model.ReplaySnapshot) error {
	w.snapshots = append(w.snapshots, snapshot)
	return nil
}
func (w *captureReplayWriter) Complete(result model.ReplayResult) error {
	w.result = result
	w.completed = true
	return nil
}
func (w *captureReplayWriter) Close() error    { return nil }
func (w *captureReplayWriter) FrameCount() int { return len(w.snapshots) }

func (m *mockExecutorService) CreateResult(ctx context.Context, res *model.Result) error {
	if m.createResultFn != nil {
		return m.createResultFn(ctx, res)
	}
	return nil
}

func (m *mockExecutorService) CalculateRankings(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.calculateRankingsFn != nil {
		return m.calculateRankingsFn(ctx, contestId)
	}
	return nil, nil
}

type mockSandbox struct {
	startSessionFn func(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error)
}

func (m *mockSandbox) ValidateBot(ctx context.Context, codePath string) error { return nil }

type mockBotSession struct{}

func (s *mockBotSession) ExecuteTurn(ctx context.Context, tick int, playerID string, perception json.RawMessage) engine.PlayerActionInput {
	return engine.PlayerActionInput{
		Status:  engine.ActionStatusValid,
		Payload: json.RawMessage(`{"thrust":"OFF","turn":"NONE","shoot":false,"shield":false}`),
	}
}

func (s *mockBotSession) Close(ctx context.Context, winner string, reason string) {}

func (m *mockSandbox) StartSession(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error) {
	if m.startSessionFn != nil {
		return m.startSessionFn(ctx, matchID, players, seed, timeout)
	}
	return &mockBotSession{}, nil
}

type mockEngineClient struct {
	startFn           func(ctx context.Context, cfg engine.StartConfig) error
	initializeMatchFn func(ctx context.Context, req engine.InitializeMatchRequest) (*engine.MatchInitializedResult, error)
	advanceTickFn     func(ctx context.Context, req engine.AdvanceTickRequest) (*engine.TickResult, error)
	finishMatchFn     func(ctx context.Context, reason string) (*engine.MatchResult, error)
	closeFn           func(ctx context.Context) error
}

func (m *mockEngineClient) Start(ctx context.Context, cfg engine.StartConfig) error {
	if m.startFn != nil {
		return m.startFn(ctx, cfg)
	}
	return nil
}

func (m *mockEngineClient) InitializeMatch(ctx context.Context, req engine.InitializeMatchRequest) (*engine.MatchInitializedResult, error) {
	if m.initializeMatchFn != nil {
		return m.initializeMatchFn(ctx, req)
	}
	return &engine.MatchInitializedResult{
		MatchID:        req.MatchID,
		InitialTick:    0,
		StateHash:      "hash-0",
		PublicSnapshot: json.RawMessage(`{"tick":0,"fighters":[],"bullets":[]}`),
		Events:         []string{"init"},
		Perceptions: map[string]json.RawMessage{
			"sub-1": json.RawMessage(`{"tick":0}`),
			"sub-2": json.RawMessage(`{"tick":0}`),
		},
	}, nil
}

func (m *mockEngineClient) AdvanceTick(ctx context.Context, req engine.AdvanceTickRequest) (*engine.TickResult, error) {
	if m.advanceTickFn != nil {
		return m.advanceTickFn(ctx, req)
	}
	return &engine.TickResult{
		Tick:           req.Tick + 1,
		Events:         []string{"tick-advanced"},
		StateHash:      "hash-1",
		IsOver:         true,
		Winner:         "sub-1",
		PublicSnapshot: json.RawMessage(`{"tick":1,"fighters":[],"bullets":[]}`),
		Perceptions: map[string]json.RawMessage{
			"sub-1": json.RawMessage(`{"tick":1}`),
			"sub-2": json.RawMessage(`{"tick":1}`),
		},
	}, nil
}

func (m *mockEngineClient) FinishMatch(ctx context.Context, reason string) (*engine.MatchResult, error) {
	if m.finishMatchFn != nil {
		return m.finishMatchFn(ctx, reason)
	}
	return &engine.MatchResult{
		FinalTick: 1,
		Reason:    reason,
		Winner:    "sub-1",
		Scores: map[string]int{
			"sub-1": 100,
			"sub-2": 50,
		},
		Rankings: []engine.PlayerRank{
			{PlayerID: "sub-1", Rank: 1, Score: 100},
			{PlayerID: "sub-2", Rank: 2, Score: 50},
		},
		FinalStateHash: "hash-1",
	}, nil
}

func (m *mockEngineClient) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

func TestMatchExecutorExecute_WithMockEngine(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	updatedStatuses := make([]string, 0)
	resultsCreated := 0
	replaySaved := false
	captured := &captureReplayWriter{}

	mockSvc := &mockExecutorService{
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
		},
		createResultFn: func(ctx context.Context, res *model.Result) error {
			resultsCreated++
			return nil
		},
		openReplayFn: func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			replaySaved = true
			return captured, nil
		},
	}

	mockSb := &mockSandbox{}

	mockEng := &mockEngineClient{}

	exec := NewMatchExecutor(mockSvc, mockSb, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		return mockEng, nil
	})
	r.NotNil(exec)

	job := &connection.MatchJob{
		JobId:         "job-101",
		Attempt:       1,
		MatchId:       "match-101",
		ContestId:     "contest-1",
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-1", "sub-2"},
		Seed:          42,
	}

	err := exec.Execute(ctx, job)
	r.NoError(err)

	r.Contains(updatedStatuses, common.MatchStatusRunning)
	r.Contains(updatedStatuses, common.MatchStatusFinished)
	r.True(replaySaved)
	r.True(captured.completed)
	r.Len(captured.snapshots, 2)
	r.Equal(2, resultsCreated)
}

func TestMatchExecutorExecute_WithFakeEngineBinary(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	// Check if bin/fake-engine exists
	if _, err := os.Stat("../../bin/fake-engine"); err != nil {
		if _, err := os.Stat("bin/fake-engine"); err != nil {
			t.Skip("bin/fake-engine binary not compiled yet")
		}
	}

	execPath := "bin/fake-engine"
	if _, err := os.Stat(execPath); err != nil {
		execPath = "../../bin/fake-engine"
	}

	updatedStatuses := make([]string, 0)
	resultsCreated := 0
	replaySaved := false
	captured := &captureReplayWriter{}

	mockSvc := &mockExecutorService{
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
		},
		createResultFn: func(ctx context.Context, res *model.Result) error {
			resultsCreated++
			return nil
		},
		openReplayFn: func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			replaySaved = true
			return captured, nil
		},
	}

	mockSb := &mockSandbox{}

	exec := NewMatchExecutor(mockSvc, mockSb, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		c := engine.NewSubprocessClient()
		err := c.Start(ctx, engine.StartConfig{
			BinaryPath: execPath,
		})
		return c, err
	})

	job := &connection.MatchJob{
		JobId:         "job-fake-1",
		Attempt:       1,
		MatchId:       "match-fake-1",
		GameId:        "starfighter",
		SubmissionIds: []string{"bot-1", "bot-2"},
		Seed:          12345,
	}

	err := exec.Execute(ctx, job)
	r.NoError(err)

	r.Contains(updatedStatuses, common.MatchStatusRunning)
	r.Contains(updatedStatuses, common.MatchStatusFinished)
	r.True(replaySaved)
	r.True(captured.completed)
	r.Equal(2, resultsCreated)
}

func TestMatchExecutorGetMatchError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	mockSvc := &mockExecutorService{
		getMatchFn: func(ctx context.Context, id string) (*model.Match, error) {
			return nil, errors.New("db error")
		},
	}

	exec := NewMatchExecutor(mockSvc, nil)
	job := &connection.MatchJob{
		JobId:   "job-err",
		MatchId: "match-err",
		GameId:  "starfighter",
	}

	err := exec.Execute(ctx, job)
	r.Error(err)
}

func TestMatchExecutorEngineStartError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	mockSvc := &mockExecutorService{}
	exec := NewMatchExecutor(mockSvc, nil, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		return nil, errors.New("cannot launch engine binary")
	})

	job := &connection.MatchJob{
		JobId:   "job-start-err",
		MatchId: "match-start-err",
		GameId:  "starfighter",
	}

	err := exec.Execute(ctx, job)
	r.Error(err)
	r.Contains(err.Error(), "cannot launch engine binary")
}

func TestMatchExecutorEngineInitializeError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	mockSvc := &mockExecutorService{}
	mockEng := &mockEngineClient{
		initializeMatchFn: func(ctx context.Context, req engine.InitializeMatchRequest) (*engine.MatchInitializedResult, error) {
			return nil, errors.New("seed invalid or game not found")
		},
	}

	exec := NewMatchExecutor(mockSvc, nil, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		return mockEng, nil
	})

	job := &connection.MatchJob{
		JobId:   "job-init-err",
		MatchId: "match-init-err",
		GameId:  "starfighter",
	}

	err := exec.Execute(ctx, job)
	r.Error(err)
	r.Contains(err.Error(), "seed invalid or game not found")
}

func TestMatchExecutorEngineTickError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	updatedStatuses := make([]string, 0)
	captured := &captureReplayWriter{}
	mockSvc := &mockExecutorService{
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
		},
		openReplayFn: func(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
			return captured, nil
		},
	}
	mockEng := &mockEngineClient{
		advanceTickFn: func(ctx context.Context, req engine.AdvanceTickRequest) (*engine.TickResult, error) {
			return nil, errors.New("tick simulation failed")
		},
	}

	exec := NewMatchExecutor(mockSvc, nil, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		return mockEng, nil
	})

	job := &connection.MatchJob{
		JobId:   "job-tick-err",
		MatchId: "match-tick-err",
		GameId:  "starfighter",
	}

	err := exec.Execute(ctx, job)
	r.Error(err)
	r.Contains(updatedStatuses, common.MatchStatusFailed)
	r.False(captured.completed, "technical failures must not be sealed as competitive replay results")
}

func TestRecordAgentIssue(t *testing.T) {
	r := require.New(t)

	issues := make(map[string]*agentIssueSummary)
	recordAgentIssue(issues, "bot-1", ErrAgentTimeout)
	recordAgentIssue(issues, "bot-1", ErrAgentInvalidAction)
	recordAgentIssue(issues, "bot-1", ErrAgentUnavailable)
	recordAgentIssue(issues, "bot-1", errors.New("other"))

	summary := issues["bot-1"]
	r.NotNil(summary)
	r.Equal(4, summary.total)
	r.Equal(1, summary.timeouts)
	r.Equal(1, summary.invalidActions)
	r.Equal(1, summary.unavailable)
	r.Equal(1, summary.execution)
}

func TestReferenceBotsComeFromStarfighterManifest(t *testing.T) {
	r := require.New(t)
	manifest := game.GetRegistry().GetManifest("starfighter")

	bot0, err := referenceBot(manifest, 0)
	r.NoError(err)
	r.Contains(bot0, "bot_hunter.py")

	bot1, err := referenceBot(manifest, 1)
	r.NoError(err)
	r.Contains(bot1, "bot_evasive.py")

	_, err = referenceBot(&game.Manifest{ID: "other"}, 0)
	r.Error(err)
}

func TestResolveAgentFallback(t *testing.T) {
	r := require.New(t)

	// Valid resolution for starfighter
	bot0, err := ResolveAgentFallback("games/starfighter", 0)
	r.NoError(err)
	r.Contains(bot0, "bot_hunter.py")

	bot1, err := ResolveAgentFallback("games/starfighter", 1)
	r.NoError(err)
	r.Contains(bot1, "bot_evasive.py")

	// Circular slot wrapping
	bot2, err := ResolveAgentFallback("games/starfighter", 2)
	r.NoError(err)
	r.Equal(bot0, bot2)

	// Non-existent directory
	_, err = ResolveAgentFallback("games/non_existent", 0)
	r.Error(err)
	r.Contains(err.Error(), "no se pudo leer manifest")

	// Manifest with no reference agents
	tmpDir := t.TempDir()
	emptyManifest := "id: empty_game\nversion: 1.0.0\n"
	r.NoError(os.WriteFile(tmpDir+"/manifest.yaml", []byte(emptyManifest), 0o644))
	_, err = ResolveAgentFallback(tmpDir, 0)
	r.Error(err)
	r.Contains(err.Error(), "no declara agentes de referencia")
}

func TestExecutor_CommitFailureDiscardsReplay(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	var discardedReplayID string
	var publishedReplayID string
	updatedStatuses := make([]string, 0)

	mockSvc := &mockExecutorService{
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
		},
		commitMatchResultFn: func(ctx context.Context, commit model.MatchResultCommit) error {
			return repository.ErrFencingTokenMismatch
		},
		discardReplayFn: func(ctx context.Context, replayID string) error {
			discardedReplayID = replayID
			return nil
		},
		publishReplayFn: func(ctx context.Context, replayID string) (*model.Replay, error) {
			publishedReplayID = replayID
			return &model.Replay{Id: replayID}, nil
		},
	}

	mockSb := &mockSandbox{}
	mockEng := &mockEngineClient{}

	exec := NewMatchExecutor(mockSvc, mockSb, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		return mockEng, nil
	})

	job := &connection.MatchJob{
		JobId:         "job-commit-fail",
		Attempt:       1,
		MatchId:       "match-commit-fail",
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-1", "sub-2"},
		Seed:          42,
		FencingToken:  1,
	}

	err := exec.Execute(ctx, job)
	r.Error(err)
	r.ErrorIs(err, repository.ErrFencingTokenMismatch)

	// Replay must be discarded and NOT published
	r.NotEmpty(discardedReplayID, "discardReplay must be called on commit failure")
	r.Empty(publishedReplayID, "publishReplay must NOT be called on commit failure")
	r.Contains(updatedStatuses, common.MatchStatusFailed)
}

func TestZombieWorkerCommitRejected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	// State tracking simulated database state for fencing
	var activeFencingToken int64 = 2 // Another worker already re-reserved job with token 2
	var committed bool

	commitFn := func(ctx context.Context, commit model.MatchResultCommit) error {
		if committed {
			return repository.ErrMatchAlreadyCommitted
		}
		if commit.FencingToken < activeFencingToken {
			return repository.ErrFencingTokenMismatch
		}
		committed = true
		return nil
	}

	var discarded1, discarded2 bool
	var published1, published2 bool

	// Worker 1 is a zombie executing with stale token 1
	mockSvc1 := &mockExecutorService{
		commitMatchResultFn: commitFn,
		discardReplayFn: func(ctx context.Context, replayID string) error {
			discarded1 = true
			return nil
		},
		publishReplayFn: func(ctx context.Context, replayID string) (*model.Replay, error) {
			published1 = true
			return &model.Replay{Id: replayID}, nil
		},
	}

	// Worker 2 is the legitimate worker executing with active token 2
	mockSvc2 := &mockExecutorService{
		commitMatchResultFn: commitFn,
		discardReplayFn: func(ctx context.Context, replayID string) error {
			discarded2 = true
			return nil
		},
		publishReplayFn: func(ctx context.Context, replayID string) (*model.Replay, error) {
			published2 = true
			return &model.Replay{Id: replayID}, nil
		},
	}

	mockSb := &mockSandbox{}
	exec1 := NewMatchExecutor(mockSvc1, mockSb, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		return &mockEngineClient{}, nil
	})
	exec2 := NewMatchExecutor(mockSvc2, mockSb, func(ctx context.Context, job *connection.MatchJob) (engine.EngineClient, error) {
		return &mockEngineClient{}, nil
	})

	// 1. Worker 1 finishes late and tries to commit with stale fencing token 1
	jobZombie := &connection.MatchJob{
		JobId:         "job-zombie",
		MatchId:       "match-zombie",
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-1", "sub-2"},
		Seed:          42,
		FencingToken:  1, // Stale!
	}
	err1 := exec1.Execute(ctx, jobZombie)
	r.Error(err1)
	r.ErrorIs(err1, repository.ErrFencingTokenMismatch)
	r.True(discarded1, "Zombie worker's temporary replay must be discarded")
	r.False(published1, "Zombie worker's replay must NOT be published")

	// 2. Worker 2 finishes with active fencing token 2
	jobLegit := &connection.MatchJob{
		JobId:         "job-zombie",
		MatchId:       "match-zombie",
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-1", "sub-2"},
		Seed:          42,
		FencingToken:  2, // Matches active token!
	}
	err2 := exec2.Execute(ctx, jobLegit)
	r.NoError(err2)
	r.False(discarded2)
	r.True(published2, "Legitimate worker's replay must be published")

	// 3. Attempting duplicate commit on already committed match
	err3 := exec2.Execute(ctx, jobLegit)
	r.Error(err3)
	r.ErrorIs(err3, repository.ErrMatchAlreadyCommitted)
}

