package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockContestRepo struct {
	repository.Repository
	listPublicContestsFn    func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	getContestFn            func(ctx context.Context, id string) (*model.Contest, error)
	getAgentFn              func(ctx context.Context, id string) (*model.Agent, error)
	hasPermissionFn         func(ctx context.Context, participantId, permission string) (bool, error)
	getRankingFn            func(ctx context.Context, contestId, agentId string) (*model.Ranking, error)
	upsertRankingFn         func(ctx context.Context, ranking *model.Ranking) error
	listRankingsByContestFn func(ctx context.Context, contestId string) ([]model.Ranking, error)
}

func (m *mockContestRepo) ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
	if m.listPublicContestsFn != nil {
		return m.listPublicContestsFn(ctx, filter)
	}
	return nil, nil
}

func (m *mockContestRepo) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	if m.getContestFn != nil {
		return m.getContestFn(ctx, id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockContestRepo) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	if m.getAgentFn != nil {
		return m.getAgentFn(ctx, id)
	}
	return nil, sql.ErrNoRows
}

func (m *mockContestRepo) HasPermission(ctx context.Context, participantId, permission string) (bool, error) {
	if m.hasPermissionFn != nil {
		return m.hasPermissionFn(ctx, participantId, permission)
	}
	return false, nil
}

func (m *mockContestRepo) GetRankingByContestAndAgent(ctx context.Context, contestId, agentId string) (*model.Ranking, error) {
	if m.getRankingFn != nil {
		return m.getRankingFn(ctx, contestId, agentId)
	}
	return nil, sql.ErrNoRows
}

func (m *mockContestRepo) UpsertRanking(ctx context.Context, ranking *model.Ranking) error {
	if m.upsertRankingFn != nil {
		return m.upsertRankingFn(ctx, ranking)
	}
	return nil
}

func (m *mockContestRepo) ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error) {
	if m.listRankingsByContestFn != nil {
		return m.listRankingsByContestFn(ctx, contestId)
	}
	return nil, nil
}

func TestListPublicContests_FilterValidation(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	t.Run("Rejects invalid state", func(t *testing.T) {
		svc := NewService(&mockContestRepo{}, nil, nil)
		filter := model.PublicContestsFilter{
			State: model.ContestState("invalid_state_xyz"),
		}

		res, err := svc.ListPublicContests(ctx, filter)
		r.Error(err)
		r.True(errors.Is(err, ErrInvalidStateFilter))
		r.Nil(res)
	})

	t.Run("Rejects draft state", func(t *testing.T) {
		svc := NewService(&mockContestRepo{}, nil, nil)
		filter := model.PublicContestsFilter{
			State: model.ContestStateDraft,
		}

		res, err := svc.ListPublicContests(ctx, filter)
		r.Error(err)
		r.True(errors.Is(err, ErrPrivateStateFilter))
		r.Nil(res)
	})

	t.Run("Accepts valid public state and forwards to repository", func(t *testing.T) {
		called := false
		now := time.Now().UTC()

		mockRepo := &mockContestRepo{
			listPublicContestsFn: func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
				called = true
				r.Equal(model.ContestStateRegistrationOpen, filter.State)
				r.True(filter.IncludeArchived)
				return []model.PublicContestSummary{
					{
						Id:          "c1",
						Name:        "Spring Contest 2026",
						Description: "Arena test",
						State:       model.ContestStateRegistrationOpen,
						StartsAt:    &now,
					},
				}, nil
			},
		}

		svc := NewService(mockRepo, nil, nil)
		filter := model.PublicContestsFilter{
			State:           model.ContestStateRegistrationOpen,
			IncludeArchived: true,
		}

		res, err := svc.ListPublicContests(ctx, filter)
		r.NoError(err)
		r.True(called)
		r.Len(res, 1)
		r.Equal("c1", res[0].Id)
		r.Equal(model.ContestStateRegistrationOpen, res[0].State)
	})

	t.Run("Empty state passes through without error", func(t *testing.T) {
		called := false
		mockRepo := &mockContestRepo{
			listPublicContestsFn: func(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
				called = true
				r.Empty(filter.State)
				r.False(filter.IncludeArchived)
				return []model.PublicContestSummary{}, nil
			},
		}

		svc := NewService(mockRepo, nil, nil)
		filter := model.PublicContestsFilter{
			State:           "",
			IncludeArchived: false,
		}

		res, err := svc.ListPublicContests(ctx, filter)
		r.NoError(err)
		r.True(called)
		r.NotNil(res)
		r.Empty(res)
	})
}

func TestGetPublicContest(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	t.Run("Returns contest when state is public", func(t *testing.T) {
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return &model.Contest{
					Id:     id,
					Name:   "Arena Cup",
					State:  model.ContestStateRegistrationOpen,
					Active: true,
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)

		c, err := svc.GetPublicContest(ctx, "arena-cup")
		r.NoError(err)
		r.NotNil(c)
		r.Equal("Arena Cup", c.Name)
	})

	t.Run("Rejects draft contest with ErrContestNotFound", func(t *testing.T) {
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return &model.Contest{
					Id:     id,
					Name:   "Secret Draft Cup",
					State:  model.ContestStateDraft,
					Active: true,
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)

		c, err := svc.GetPublicContest(ctx, "draft-cup")
		r.Error(err)
		r.True(errors.Is(err, ErrContestNotFound))
		r.Nil(c)
	})

	t.Run("Rejects inactive contest", func(t *testing.T) {
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return &model.Contest{
					Id:     id,
					Name:   "Inactive Cup",
					State:  model.ContestStatePublished,
					Active: false,
				}, nil
			},
		}
		svc := NewService(repo, nil, nil)

		c, err := svc.GetPublicContest(ctx, "inactive-cup")
		r.Error(err)
		r.True(errors.Is(err, ErrContestNotFound))
		r.Nil(c)
	})
}

func TestEnrollAgent(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	openContest := &model.Contest{
		Id:     "contest-1",
		GameId: "arena-basica",
		State:  model.ContestStateRegistrationOpen,
		Active: true,
	}

	finishedContest := &model.Contest{
		Id:     "contest-2",
		GameId: "arena-basica",
		State:  model.ContestStateFinished,
		Active: true,
	}

	validAgent := &model.Agent{
		Id:            "agent-1",
		ParticipantId: "part-1",
		GameId:        "arena-basica",
		Active:        true,
	}

	diffGameAgent := &model.Agent{
		Id:            "agent-2",
		ParticipantId: "part-1",
		GameId:        "other-game",
		Active:        true,
	}

	t.Run("Successful enrollment", func(t *testing.T) {
		var savedRanking *model.Ranking
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return openContest, nil
			},
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return validAgent, nil
			},
			getRankingFn: func(ctx context.Context, contestId, agentId string) (*model.Ranking, error) {
				return nil, sql.ErrNoRows
			},
			upsertRankingFn: func(ctx context.Context, ranking *model.Ranking) error {
				savedRanking = ranking
				return nil
			},
		}
		svc := NewService(repo, nil, nil)

		ranking, err := svc.EnrollAgent(ctx, "part-1", "contest-1", "agent-1")
		r.NoError(err)
		r.NotNil(ranking)
		r.Equal("contest-1", ranking.ContestId)
		r.Equal("agent-1", ranking.AgentId)
		r.Equal("part-1", ranking.ParticipantId)
		r.Equal(0, ranking.Score)
		r.Equal(1, ranking.Rank)
		r.NotNil(savedRanking)
	})

	t.Run("Fails if contest is not open for registration", func(t *testing.T) {
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return finishedContest, nil
			},
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return validAgent, nil
			},
		}
		svc := NewService(repo, nil, nil)

		ranking, err := svc.EnrollAgent(ctx, "part-1", "contest-2", "agent-1")
		r.Error(err)
		r.True(errors.Is(err, ErrRegistrationClosed))
		r.Nil(ranking)
	})

	t.Run("Fails if agent belongs to another participant", func(t *testing.T) {
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return openContest, nil
			},
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return validAgent, nil
			},
			hasPermissionFn: func(ctx context.Context, participantId, permission string) (bool, error) {
				return false, nil
			},
		}
		svc := NewService(repo, nil, nil)

		ranking, err := svc.EnrollAgent(ctx, "different-part", "contest-1", "agent-1")
		r.Error(err)
		r.True(errors.Is(err, ErrUnauthorizedAgent))
		r.Nil(ranking)
	})

	t.Run("Fails if game does not match contest", func(t *testing.T) {
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return openContest, nil
			},
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return diffGameAgent, nil
			},
		}
		svc := NewService(repo, nil, nil)

		ranking, err := svc.EnrollAgent(ctx, "part-1", "contest-1", "agent-2")
		r.Error(err)
		r.True(errors.Is(err, ErrGameMismatch))
		r.Nil(ranking)
	})

	t.Run("Fails if agent already enrolled", func(t *testing.T) {
		repo := &mockContestRepo{
			getContestFn: func(ctx context.Context, id string) (*model.Contest, error) {
				return openContest, nil
			},
			getAgentFn: func(ctx context.Context, id string) (*model.Agent, error) {
				return validAgent, nil
			},
			getRankingFn: func(ctx context.Context, contestId, agentId string) (*model.Ranking, error) {
				return &model.Ranking{Id: "existing-ranking"}, nil
			},
		}
		svc := NewService(repo, nil, nil)

		ranking, err := svc.EnrollAgent(ctx, "part-1", "contest-1", "agent-1")
		r.Error(err)
		r.True(errors.Is(err, ErrAgentAlreadyEnrolled))
		r.Nil(ranking)
	})
}
