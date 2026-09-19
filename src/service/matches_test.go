package service

import (
	"context"
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
	return &model.Match{Id: id, GameId: "starfighter", Status: common.MatchStatusPending}, nil
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

	// CreateMatch with submissions (should enqueue)
	newMatch := &model.Match{GameId: "starfighter", ContestId: "c1"}
	err = svc.CreateMatch(ctx, newMatch, []string{"sub-1", "sub-2"})
	r.NoError(err)
	r.NotEmpty(newMatch.Id)
	r.Equal(common.MatchStatusPending, newMatch.Status)
	r.NotZero(newMatch.Seed)
	r.Equal(1, queue.Len())

	// RunMatch
	err = svc.RunMatch(ctx, "m1")
	r.NoError(err)
	r.Equal(2, queue.Len())

	// UpdateMatch & ActivateMatch
	r.NoError(svc.UpdateMatch(ctx, newMatch))
	r.NoError(svc.ActivateMatch(ctx, newMatch.Id, false))
	r.ErrorIs(svc.CreateMatch(ctx, &model.Match{GameId: "other-game"}, nil), ErrUnsupportedGame)
}
