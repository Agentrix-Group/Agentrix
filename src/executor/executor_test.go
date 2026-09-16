package executor

import (
	"context"
	"errors"
	"testing"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/connection"
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
	executeTurnFn      func(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error)
	filterPerceptionFn func(state *game.GameState, playerID string) SlotPerception
}

func (m *mockSandbox) ExecuteTurn(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error) {
	if m.executeTurnFn != nil {
		return m.executeTurnFn(ctx, codePath, state, playerID)
	}
	return game.Action{Type: game.ActionRest}, nil
}

func (m *mockSandbox) FilterPerception(state *game.GameState, playerID string) SlotPerception {
	if m.filterPerceptionFn != nil {
		return m.filterPerceptionFn(state, playerID)
	}
	return SlotPerception{}
}

func TestMatchExecutorExecute(t *testing.T) {
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
		executeTurnFn: func(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error) {
			return game.Action{Type: game.ActionRest}, nil
		},
	}

	exec := NewMatchExecutor(mockSvc, mockSb)
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
