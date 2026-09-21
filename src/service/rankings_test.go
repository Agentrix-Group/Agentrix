package service

import (
	"context"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockRankingsRepo struct {
	repository.Repository
	getContestFn                func(ctx context.Context, id string) (*model.Contest, error)
	listContestEntriesFn        func(ctx context.Context, contestId string) ([]model.ContestEntry, error)
	listRankingsFn              func(ctx context.Context) ([]model.Ranking, error)
	listRankingsByContestFn     func(ctx context.Context, contestId string) ([]model.Ranking, error)
	getRankingFn                func(ctx context.Context, id string) (*model.Ranking, error)
	getRankingByContestAndAgent func(ctx context.Context, contestId, agentId string) (*model.Ranking, error)
	listMatchesByContestFn      func(ctx context.Context, contestId string) ([]model.Match, error)
	listResultsByMatchFn        func(ctx context.Context, matchId string) ([]model.Result, error)
	getSubmissionFn             func(ctx context.Context, id string) (*model.Submission, error)
	getAgentFn                  func(ctx context.Context, id string) (*model.Agent, error)
	upsertRankingFn             func(ctx context.Context, ranking *model.Ranking) error
	clearAppliedRunsFn          func(ctx context.Context, contestId string) error
	recordAppliedRunFn          func(ctx context.Context, contestId, runId, matchId string) error
	isRunAppliedToRankingFn     func(ctx context.Context, contestId, runId string) (bool, error)
	createRankingSnapshotFn     func(ctx context.Context, snapshot *model.RankingSnapshot) error
	listRankingSnapshotsFn      func(ctx context.Context, contestId string) ([]model.RankingSnapshot, error)
	getLatestRankingSnapshotFn  func(ctx context.Context, contestId string) (*model.RankingSnapshot, error)
	getRankingSnapshotByVerFn   func(ctx context.Context, contestId string, version int) (*model.RankingSnapshot, error)
}

func (m *mockRankingsRepo) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	if m.getContestFn != nil {
		return m.getContestFn(ctx, id)
	}
	return &model.Contest{Id: id, Name: "Test Contest"}, nil
}

func (m *mockRankingsRepo) ListContestEntries(ctx context.Context, contestId string) ([]model.ContestEntry, error) {
	if m.listContestEntriesFn != nil {
		return m.listContestEntriesFn(ctx, contestId)
	}
	return nil, nil
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

func (m *mockRankingsRepo) GetRankingByContestAndAgent(ctx context.Context, contestId, agentId string) (*model.Ranking, error) {
	if m.getRankingByContestAndAgent != nil {
		return m.getRankingByContestAndAgent(ctx, contestId, agentId)
	}
	return nil, nil
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
	return &model.Agent{Id: id, OwnerUserId: "user-1"}, nil
}

func (m *mockRankingsRepo) UpsertRanking(ctx context.Context, ranking *model.Ranking) error {
	if m.upsertRankingFn != nil {
		return m.upsertRankingFn(ctx, ranking)
	}
	return nil
}

func (m *mockRankingsRepo) ClearAppliedRuns(ctx context.Context, contestId string) error {
	if m.clearAppliedRunsFn != nil {
		return m.clearAppliedRunsFn(ctx, contestId)
	}
	return nil
}

func (m *mockRankingsRepo) RecordAppliedRun(ctx context.Context, contestId, runId, matchId string) error {
	if m.recordAppliedRunFn != nil {
		return m.recordAppliedRunFn(ctx, contestId, runId, matchId)
	}
	return nil
}

func (m *mockRankingsRepo) IsRunAppliedToRanking(ctx context.Context, contestId, runId string) (bool, error) {
	if m.isRunAppliedToRankingFn != nil {
		return m.isRunAppliedToRankingFn(ctx, contestId, runId)
	}
	return false, nil
}

func (m *mockRankingsRepo) CreateRankingSnapshot(ctx context.Context, snapshot *model.RankingSnapshot) error {
	if m.createRankingSnapshotFn != nil {
		return m.createRankingSnapshotFn(ctx, snapshot)
	}
	return nil
}

func (m *mockRankingsRepo) ListRankingSnapshots(ctx context.Context, contestId string) ([]model.RankingSnapshot, error) {
	if m.listRankingSnapshotsFn != nil {
		return m.listRankingSnapshotsFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockRankingsRepo) GetLatestRankingSnapshot(ctx context.Context, contestId string) (*model.RankingSnapshot, error) {
	if m.getLatestRankingSnapshotFn != nil {
		return m.getLatestRankingSnapshotFn(ctx, contestId)
	}
	return nil, nil
}

func (m *mockRankingsRepo) GetRankingSnapshotByVersion(ctx context.Context, contestId string, version int) (*model.RankingSnapshot, error) {
	if m.getRankingSnapshotByVerFn != nil {
		return m.getRankingSnapshotByVerFn(ctx, contestId, version)
	}
	return nil, nil
}

func TestRankingsService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	now := time.Now().UTC()
	committedRun := "run-1"

	storage := make(map[string]*model.Ranking)

	mockRepo := &mockRankingsRepo{
		getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
			pol := model.DefaultScoringPolicy()
			return &model.Contest{
				Id:            id,
				Name:          "Tournament 1",
				ScoringPolicy: &pol,
			}, nil
		},
		getRankingByContestAndAgent: func(ctx context.Context, contestId, agentId string) (*model.Ranking, error) {
			if rk, ok := storage[contestId+":"+agentId]; ok {
				return rk, nil
			}
			return nil, nil
		},
		upsertRankingFn: func(ctx context.Context, ranking *model.Ranking) error {
			cp := *ranking
			storage[ranking.ContestId+":"+ranking.AgentId] = &cp
			return nil
		},
		listRankingsFn: func(ctx context.Context) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r1", Score: 100}}, nil
		},
		listRankingsByContestFn: func(ctx context.Context, contestId string) ([]model.Ranking, error) {
			return []model.Ranking{{Id: "r1", ContestId: contestId, Score: 100}}, nil
		},
		listMatchesByContestFn: func(ctx context.Context, contestId string) ([]model.Match, error) {
			return []model.Match{
				{
					Id:             "m1",
					ContestId:      contestId,
					Status:         "finished",
					CommittedRunId: &committedRun,
					FinishedAt:     &now,
				},
			}, nil
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
			return &model.Agent{Id: id, OwnerUserId: "user-" + id}, nil
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

	// CalculateRankings / RecalculateContestRankings
	calculated, err := svc.CalculateRankings(ctx, "c1")
	r.NoError(err)
	r.Len(calculated, 2)
	r.Equal("agent-1", calculated[0].AgentId)
	r.Equal(1, calculated[0].Rank)
	r.Equal(3, calculated[0].Points) // 3 win points
	r.Equal(100, calculated[0].Score)
	r.Equal(1, calculated[0].Wins)

	r.Equal("agent-2", calculated[1].AgentId)
	r.Equal(2, calculated[1].Rank)
	r.Equal(0, calculated[1].Points) // 0 loss points
	r.Equal(50, calculated[1].Score)
	r.Equal(1, calculated[1].Losses)

	// Recalculating produces 100% identical results (same IDs, ranks, points)
	recalculated, err := svc.RecalculateContestRankings(ctx, "c1")
	r.NoError(err)
	r.Equal(calculated[0].Id, recalculated[0].Id)
	r.Equal(calculated[0].Rank, recalculated[0].Rank)
	r.Equal(calculated[0].Points, recalculated[0].Points)
	r.Equal(calculated[1].Id, recalculated[1].Id)
	r.Equal(calculated[1].Rank, recalculated[1].Rank)
	r.Equal(calculated[1].Points, recalculated[1].Points)
}

func TestPublishRankingSnapshot(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	var createdSnapshot *model.RankingSnapshot
	mockRepo := &mockRankingsRepo{
		getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
			return &model.Contest{Id: id, Name: "Championship"}, nil
		},
		listRankingsByContestFn: func(ctx context.Context, contestId string) ([]model.Ranking, error) {
			return []model.Ranking{
				{AgentId: "agent-1", Rank: 1, Points: 6},
				{AgentId: "agent-2", Rank: 2, Points: 3},
			}, nil
		},
		getLatestRankingSnapshotFn: func(ctx context.Context, contestId string) (*model.RankingSnapshot, error) {
			return &model.RankingSnapshot{Version: 1}, nil
		},
		createRankingSnapshotFn: func(ctx context.Context, snapshot *model.RankingSnapshot) error {
			createdSnapshot = snapshot
			return nil
		},
	}

	svc := NewService(mockRepo, nil, nil)
	snap, err := svc.PublishRankingSnapshot(ctx, "contest-1", "user-admin")
	r.NoError(err)
	r.NotNil(snap)
	r.Equal(2, snap.Version) // Increment from 1 to 2
	r.Equal("contest-1", snap.ContestId)
	r.Equal("user-admin", *snap.PublishedBy)
	r.Len(snap.Rankings, 2)
	r.Equal(createdSnapshot, snap)
}
