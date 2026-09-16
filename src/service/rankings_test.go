package service

import (
	"context"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockRankingsRepo struct {
	repository.Repository
	listRankingsFn          func(ctx context.Context) ([]model.Ranking, error)
	listRankingsByContestFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
	getRankingFn            func(ctx context.Context, id string) (*model.Ranking, error)
	listMatchesByContestFn  func(ctx context.Context, contestId string) ([]model.Match, error)
	listResultsByMatchFn    func(ctx context.Context, matchId string) ([]model.Result, error)
	getSubmissionFn         func(ctx context.Context, id string) (*model.Submission, error)
	getAgentFn              func(ctx context.Context, id string) (*model.Agent, error)
	upsertRankingFn         func(ctx context.Context, ranking *model.Ranking) error
}

func (m *mockRankingsRepo) ListRankings(ctx context.Context) ([]model.Ranking, error) {
	if m.listRankingsFn != nil {
		return m.listRankingsFn(ctx)
	}
	return nil, nil
}

func (m *mockRankingsRepo) ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.listRankingsByContestFn != nil {
		return m.listRankingsByContestFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockRankingsRepo) GetRanking(ctx context.Context, id string) (*model.Ranking, error) {
	if m.getRankingFn != nil {
		return m.getRankingFn(ctx, id)
	}
	return &model.Ranking{Id: id, Score: 100}, nil
}

func (m *mockRankingsRepo) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	if m.listMatchesByContestFn != nil {
		return m.listMatchesByContestFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockRankingsRepo) ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error) {
	if m.listResultsByMatchFn != nil {
		return m.listResultsByMatchFn(ctx, matchId)
	}
	return nil, nil
}

func (m *mockRankingsRepo) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	if m.getSubmissionFn != nil {
		return m.getSubmissionFn(ctx, id)
	}
	return &model.Submission{Id: id, AgentId: "agent-1"}, nil
}

func (m *mockRankingsRepo) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	if m.getAgentFn != nil {
		return m.getAgentFn(ctx, id)
	}
	return &model.Agent{Id: id, ParticipantId: "part-1"}, nil
}

func (m *mockRankingsRepo) UpsertRanking(ctx context.Context, ranking *model.Ranking) error {
	if m.upsertRankingFn != nil {
		return m.upsertRankingFn(ctx, ranking)
	}
	return nil
}

func TestRankingsService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	mockRepo := &mockRankingsRepo{
		listRankingsFn: func(ctx context.Context) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r1", Score: 100}}, nil
		},
		listRankingsByContestFn: func(ctx context.Context, contestId string) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r1", ContestId: contestId, Score: 100}}, nil
		},
		listMatchesByContestFn: func(ctx context.Context, contestId string) ([]model.Match, error) {
			return []model.Match{{Id: "m1", ContestId: contestId}}, nil
		},
		listResultsByMatchFn: func(ctx context.Context, matchId string) ([]model.Result, error) {
			return []model.Result{
				{Id: "res-1", MatchId: matchId, SubmissionId: "sub-1", Score: 100, Rank: 1},
				{Id: "res-2", MatchId: matchId, SubmissionId: "sub-2", Score: 50, Rank: 2},
			}, nil
		},
		getSubmissionFn: func(ctx context.Context, id string) (*model.Submission, error) {
			if id == "sub-1" {
				return &model.Submission{Id: id, AgentId: "agent-1"}, nil
			}
			return &model.Submission{Id: id, AgentId: "agent-2"}, nil
		},
		getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
			return &model.Agent{Id: id, ParticipantId: "part-" + id}, nil
		},
	}

	svc := NewService(mockRepo, nil, nil)

	// ListRankings
	rankings, err := svc.ListRankings(ctx)
	r.NoError(err)
	r.Len(rankings, 1)

	// ListRankingsByContest
	contestRanks, err := svc.ListRankingsByContest(ctx, "c1")
	r.NoError(err)
	r.Len(contestRanks, 1)

	// GetRanking
	rank, err := svc.GetRanking(ctx, "r1")
	r.NoError(err)
	r.Equal("r1", rank.Id)

	// CalculateRankings
	calculated, err := svc.CalculateRankings(ctx, "c1")
	r.NoError(err)
	r.Len(calculated, 2)
	r.Equal("agent-1", calculated[0].AgentId)
	r.Equal(1, calculated[0].Rank)
	r.Equal(100, calculated[0].Score)
	r.Equal("agent-2", calculated[1].AgentId)
	r.Equal(2, calculated[1].Rank)
	r.Equal(50, calculated[1].Score)
}
