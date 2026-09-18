package executor

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/engine"
	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

type mockExecutorService struct {
	service.Service
	getMatchFn          func(ctx context.Context, id string) (*model.Match, error)
	updateMatchFn       func(ctx context.Context, match *model.Match) error
	getSubmissionFn     func(ctx context.Context, id string) (*model.Submission, error)
	saveReplayFn        func(ctx context.Context, replay *model.Replay, data *model.ReplayData) error
	createResultFn      func(ctx context.Context, res *model.Result) error
	calculateRankingsFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
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

func (m *mockExecutorService) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	if m.getSubmissionFn != nil {
		return m.getSubmissionFn(ctx, id)
	}
	return &model.Submission{Id: id, CodePath: "fake.py"}, nil
}

func (m *mockExecutorService) SaveReplay(ctx context.Context, replay *model.Replay, data *model.ReplayData) error {
	if m.saveReplayFn != nil {
		return m.saveReplayFn(ctx, replay, data)
	}
	return nil
}

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
	executeTurnFn               func(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error)
	executeTurnWithPerceptionFn func(ctx context.Context, codePath string, perception interface{}, playerID string) (game.Action, error)
	filterPerceptionFn          func(state *game.GameState, playerID string) SlotPerception
	startSessionFn              func(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error)
}

// mockBotSession bridges the session-based interface to
// mockSandbox.executeTurnWithPerceptionFn, so existing tests written
// against the old per-call sandbox API keep exercising the same
// expectations without needing to know about persistent sessions.
type mockBotSession struct {
	sandbox *mockSandbox
	players map[string]string
}

func (s *mockBotSession) ExecuteTurn(ctx context.Context, tick int, playerID string, perception interface{}) engine.PlayerActionInput {
	action, err := s.sandbox.ExecuteTurnWithPerception(ctx, s.players[playerID], perception, playerID)
	if err != nil {
		return engine.PlayerActionInput{Status: engine.ActionStatusTimeout, ErrorDetails: err.Error()}
	}
	return engine.PlayerActionInput{Status: engine.ActionStatusValid, ActionType: string(action.Type), Payload: action.Payload}
}

func (s *mockBotSession) Close(ctx context.Context, winner string, reason string) {}

func (m *mockSandbox) ExecuteTurn(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error) {
	if m.executeTurnFn != nil {
		return m.executeTurnFn(ctx, codePath, state, playerID)
	}
	return game.Action{Type: game.ActionRest}, nil
}

func (m *mockSandbox) ExecuteTurnWithPerception(ctx context.Context, codePath string, perception interface{}, playerID string) (game.Action, error) {
	if m.executeTurnWithPerceptionFn != nil {
		return m.executeTurnWithPerceptionFn(ctx, codePath, perception, playerID)
	}
	return game.Action{Type: game.ActionRest}, nil
}

func (m *mockSandbox) FilterPerception(state *game.GameState, playerID string) SlotPerception {
	if m.filterPerceptionFn != nil {
		return m.filterPerceptionFn(state, playerID)
	}
	return SlotPerception{}
}

func (m *mockSandbox) StartSession(ctx context.Context, matchID string, players map[string]string, seed int64, timeout time.Duration) (BotSession, error) {
	if m.startSessionFn != nil {
		return m.startSessionFn(ctx, matchID, players, seed, timeout)
	}
	return &mockBotSession{sandbox: m, players: players}, nil
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
		MatchID:     req.MatchID,
		InitialTick: 0,
		StateHash:   "hash-0",
		Events:      []string{"init"},
		Perceptions: map[string]map[string]interface{}{
			"sub-1": {"tick": 0},
			"sub-2": {"tick": 0},
		},
	}, nil
}

func (m *mockEngineClient) AdvanceTick(ctx context.Context, req engine.AdvanceTickRequest) (*engine.TickResult, error) {
	if m.advanceTickFn != nil {
		return m.advanceTickFn(ctx, req)
	}
	return &engine.TickResult{
		Tick:      req.Tick,
		Events:    []string{"tick-advanced"},
		StateHash: "hash-1",
		IsOver:    true,
		Winner:    "sub-1",
		Perceptions: map[string]map[string]interface{}{
			"sub-1": {"tick": req.Tick},
			"sub-2": {"tick": req.Tick},
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
		FinalStateHash: "hash-final",
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

	mockSvc := &mockExecutorService{
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
		},
		createResultFn: func(ctx context.Context, res *model.Result) error {
			resultsCreated++
			return nil
		},
		saveReplayFn: func(ctx context.Context, replay *model.Replay, data *model.ReplayData) error {
			replaySaved = true
			return nil
		},
	}

	mockSb := &mockSandbox{
		executeTurnWithPerceptionFn: func(ctx context.Context, codePath string, perception interface{}, playerID string) (game.Action, error) {
			return game.Action{Type: game.ActionRest}, nil
		},
	}

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
		GameId:        "arena-basica",
		SubmissionIds: []string{"sub-1", "sub-2"},
		Seed:          42,
	}

	err := exec.Execute(ctx, job)
	r.NoError(err)

	r.Contains(updatedStatuses, common.MatchStatusRunning)
	r.Contains(updatedStatuses, common.MatchStatusFinished)
	r.True(replaySaved)
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

	mockSvc := &mockExecutorService{
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
		},
		createResultFn: func(ctx context.Context, res *model.Result) error {
			resultsCreated++
			return nil
		},
		saveReplayFn: func(ctx context.Context, replay *model.Replay, data *model.ReplayData) error {
			replaySaved = true
			return nil
		},
	}

	mockSb := &mockSandbox{
		executeTurnWithPerceptionFn: func(ctx context.Context, codePath string, perception interface{}, playerID string) (game.Action, error) {
			return game.Action{Type: game.ActionUp}, nil
		},
	}

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
		GameId:        "arena-basica",
		SubmissionIds: []string{"bot-1", "bot-2"},
		Seed:          12345,
	}

	err := exec.Execute(ctx, job)
	r.NoError(err)

	r.Contains(updatedStatuses, common.MatchStatusRunning)
	r.Contains(updatedStatuses, common.MatchStatusFinished)
	r.True(replaySaved)
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
		GameId:  "arena-basica",
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
		GameId:  "arena-basica",
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
		GameId:  "arena-basica",
	}

	err := exec.Execute(ctx, job)
	r.Error(err)
	r.Contains(err.Error(), "seed invalid or game not found")
}

func TestMatchExecutorEngineTickError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	updatedStatuses := make([]string, 0)
	mockSvc := &mockExecutorService{
		updateMatchFn: func(ctx context.Context, match *model.Match) error {
			updatedStatuses = append(updatedStatuses, match.Status)
			return nil
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
		GameId:  "arena-basica",
	}

	err := exec.Execute(ctx, job)
	r.Error(err)
	r.Contains(updatedStatuses, common.MatchStatusFailed)
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
