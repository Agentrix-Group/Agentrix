package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockMatchesRepo struct {
	repository.Repository
	listMatchesFn          func(ctx context.Context) ([]model.Match, error)
	listMatchesByContestFn func(ctx context.Context, contestId string) ([]model.Match, error)
	getMatchFn             func(ctx context.Context, id string) (*model.Match, error)
	createMatchFn          func(ctx context.Context, match *model.Match) error
	updateMatchFn          func(ctx context.Context, match *model.Match) error
	activateMatchFn        func(ctx context.Context, id string, isActive bool) error
	listResultsByMatchFn   func(ctx context.Context, matchId string) ([]model.Result, error)
}

func (m *mockMatchesRepo) ListMatches(ctx context.Context) ([]model.Match, error) {
	if m.listMatchesFn != nil {
		return m.listMatchesFn(ctx)
	}
	return nil, nil
}

func (m *mockMatchesRepo) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	if m.listMatchesByContestFn != nil {
		return m.listMatchesByContestFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockMatchesRepo) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	if m.getMatchFn != nil {
		return m.getMatchFn(ctx, id)
	}
	return &model.Match{
		Id:     id,
		GameId: "starfighter",
		Status: common.MatchStatusPending,
		Slots: []model.MatchSlot{
			{SlotIndex: 0, SubmissionId: "sub-1"},
			{SlotIndex: 1, SubmissionId: "sub-2"},
		},
	}, nil
}

func (m *mockMatchesRepo) CreateMatch(ctx context.Context, match *model.Match) error {
	if m.createMatchFn != nil {
		return m.createMatchFn(ctx, match)
	}
	return nil
}

func (m *mockMatchesRepo) UpdateMatch(ctx context.Context, match *model.Match) error {
	if m.updateMatchFn != nil {
		return m.updateMatchFn(ctx, match)
	}
	return nil
}

func (m *mockMatchesRepo) ActivateMatch(ctx context.Context, id string, isActive bool) error {
	if m.activateMatchFn != nil {
		return m.activateMatchFn(ctx, id, isActive)
	}
	return nil
}

func (m *mockMatchesRepo) ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error) {
	if m.listResultsByMatchFn != nil {
		return m.listResultsByMatchFn(ctx, matchId)
	}
	return nil, nil
}

func (m *mockMatchesRepo) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	return &model.Contest{Id: id, State: model.ContestStateRegistrationOpen, Active: true}, nil
}

func (m *mockMatchesRepo) GetContestEntryBySubmission(ctx context.Context, contestId, submissionId string) (*model.ContestEntry, error) {
	return &model.ContestEntry{
		Id:           "ce-1",
		ContestId:    contestId,
		SubmissionId: submissionId,
		Status:       model.ContestEntryStatusEnrolled,
	}, nil
}

func (m *mockMatchesRepo) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	return &model.Submission{Id: id, Active: true, Status: "ready"}, nil
}

func (m *mockMatchesRepo) UpdateMatchStatusCAS(ctx context.Context, matchId string, expectedStatus string, newStatus model.MatchStatus) (bool, error) {
	return true, nil
}

func (m *mockMatchesRepo) CreateMatchRun(ctx context.Context, run *model.MatchRun) error {
	return nil
}

func (m *mockMatchesRepo) ListMatchRunsByMatch(ctx context.Context, matchId string) ([]*model.MatchRun, error) {
	return nil, nil
}

func (m *mockMatchesRepo) GetLatestMatchRunByMatch(ctx context.Context, matchId string) (*model.MatchRun, error) {
	return nil, nil
}

func TestMatchesService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	mockRepo := &mockMatchesRepo{
		listMatchesFn: func(ctx context.Context) ([]model.Match, error) {
			return []model.Match{{Id: "m1", GameId: "starfighter"}}, nil
		},
		listMatchesByContestFn: func(ctx context.Context, contestId string) ([]model.Match, error) {
			return []model.Match{{Id: "m1", ContestId: contestId}}, nil
		},
		listResultsByMatchFn: func(ctx context.Context, matchId string) ([]model.Result, error) {
			return []model.Result{{Id: "res-1", MatchId: matchId, SubmissionId: "sub-1"}}, nil
		},
	}

	queue := connection.NewJobQueue(5)
	defer queue.Close()

	svc := NewService(mockRepo, nil, queue)

	// ListMatches
	matches, err := svc.ListMatches(ctx)
	r.NoError(err)
	r.Len(matches, 1)

	// ListMatchesByContest
	contestMatches, err := svc.ListMatchesByContest(ctx, "c1")
	r.NoError(err)
	r.Len(contestMatches, 1)

	// GetMatch (should include results)
	match, err := svc.GetMatch(ctx, "m1")
	r.NoError(err)
	r.Equal("m1", match.Id)
	r.Len(match.Results, 1)

	// CreateMatch with submissions (should schedule, not enqueue directly)
	newMatch := &model.Match{GameId: "starfighter", ContestId: "c1"}
	resp, err := svc.CreateMatch(ctx, newMatch, []string{"sub-1", "sub-2"})
	r.NoError(err)
	r.NotNil(resp)
	r.NotEmpty(newMatch.Id)
	r.Equal(newMatch.Id, resp.MatchId)
	r.Equal(common.MatchStatusPending, newMatch.Status)
	r.NotZero(newMatch.Seed)
	r.Len(resp.Slots, 2)
	r.Equal(0, queue.Len(), "CreateMatch schedules but does not enqueue")

	// RunMatch fails closed when no engine digest is configured
	_, err = svc.RunMatch(ctx, newMatch.Id)
	r.ErrorIs(err, model.ErrInvalidExecutionSpec)
	r.ErrorContains(err, "engine binary digest is required (fail closed)")

	// Now configure valid engine digest and verify successful enqueue
	t.Setenv("AGENTRIX_ENGINE_DIGEST", strings.Repeat("a", 64))
	runResp, err := svc.RunMatch(ctx, newMatch.Id)
	r.NoError(err)
	r.NotNil(runResp)
	r.Equal(newMatch.Id, runResp.MatchId)
	r.Equal("queued", runResp.Status)
	r.NotEmpty(runResp.RunId)
	r.NotEmpty(runResp.JobId)
	r.Equal(1, queue.Len(), "RunMatch enqueued the job")

	// UpdateMatch & ActivateMatch
	r.NoError(svc.UpdateMatch(ctx, newMatch))
	r.NoError(svc.ActivateMatch(ctx, newMatch.Id, false))
	_, err = svc.CreateMatch(ctx, &model.Match{GameId: "other-game"}, nil)
	r.ErrorIs(err, ErrUnsupportedGame)
}

func TestValidateMatchParticipants(t *testing.T) {
	r := require.New(t)
	r.NoError(validateMatchParticipants([]string{"a", "b"}))
	r.NoError(validateMatchParticipants([]string{"a", "b", "c", "d", "e"}))
	for name, ids := range map[string][]string{
		"none":      nil,
		"one":       {"a"},
		"six":       {"a", "b", "c", "d", "e", "f"},
		"duplicate": {"a", "b", "a"},
		"empty id":  {"a", ""},
	} {
		r.ErrorIs(validateMatchParticipants(ids), ErrInvalidParticipants, name)
	}
}
