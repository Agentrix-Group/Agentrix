package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

var (
	ErrInvalidStateFilter   = errors.New("the specified state filter is invalid")
	ErrPrivateStateFilter   = errors.New("the specified state is private and cannot be queried publicly")
	ErrContestNotFound      = errors.New("contest not found")
	ErrRegistrationClosed   = errors.New("contest is not currently open for registration")
	ErrAgentNotFound        = errors.New("agent not found")
	ErrUnauthorizedAgent    = errors.New("agent does not belong to the authenticated participant")
	ErrGameMismatch         = errors.New("agent is not configured for the contest's game")
	ErrAgentAlreadyEnrolled = errors.New("agent is already enrolled in this contest")
)

// ContestService defines contest use-case operations for consumer segregation (ATD-015).
type ContestService interface {
	ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	GetPublicContest(ctx context.Context, id string) (*model.Contest, error)
	ListContests(ctx context.Context) ([]model.Contest, error)
	GetContest(ctx context.Context, id string) (*model.Contest, error)
	CreateContest(ctx context.Context, contest *model.Contest) error
	UpdateContest(ctx context.Context, contest *model.Contest) error
	ActivateContest(ctx context.Context, id string, isActive bool) error
	EnrollAgent(ctx context.Context, participantId string, contestId string, agentId string) (*model.Ranking, error)
	ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error)
}

func (s *service) ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
	tracer.Debugf(ctx, "Listing public contests with filter state='%s', include_archived=%t", filter.State, filter.IncludeArchived)

	if filter.State != "" {
		if !filter.State.IsValid() {
			tracer.Warnf(ctx, "Rejected query for unrecognized state filter: %s", filter.State)
			return nil, ErrInvalidStateFilter
		}
		if !filter.State.IsPublic() {
			tracer.Warnf(ctx, "Rejected public query for private state filter: %s", filter.State)
			return nil, ErrPrivateStateFilter
		}
	}

	return s.repo.ListPublicContests(ctx, filter)
}

func (s *service) ListContests(ctx context.Context) ([]model.Contest, error) {
	return s.repo.ListContests(ctx)
}

func (s *service) GetContest(ctx context.Context, id string) (*model.Contest, error) {
	return s.repo.GetContest(ctx, id)
}

func (s *service) CreateContest(ctx context.Context, contest *model.Contest) error {
	tracer.Debugf(ctx, "Creating contest '%s'", contest.Name)
	if contest.Id == "" {
		contest.Id = uuid.New().String()
	}
	if contest.Status == "" {
		contest.Status = "upcoming"
	}
	contest.Active = true
	contest.CreatedAt = time.Now().UTC()

	return s.repo.CreateContest(ctx, contest)
}

func (s *service) UpdateContest(ctx context.Context, contest *model.Contest) error {
	return s.repo.UpdateContest(ctx, contest)
}

func (s *service) ActivateContest(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateContest(ctx, id, isActive)
}

func (s *service) GetPublicContest(ctx context.Context, id string) (*model.Contest, error) {
	tracer.Debugf(ctx, "Querying public contest details for '%s'", id)
	c, err := s.repo.GetContest(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	if !c.Active || !c.State.IsPublic() {
		tracer.Warnf(ctx, "Rejected public query for contest '%s': not active or in non-public state '%s'", id, c.State)
		return nil, ErrContestNotFound
	}

	return c, nil
}

func (s *service) EnrollAgent(ctx context.Context, participantId string, contestId string, agentId string) (*model.Ranking, error) {
	tracer.Debugf(ctx, "Enrolling agent '%s' into contest '%s' for participant '%s'", agentId, contestId, participantId)

	contest, err := s.repo.GetContest(ctx, contestId)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Enrollment failed: contest '%s' not found", contestId)
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	if !contest.Active {
		tracer.Warnf(ctx, "Enrollment failed: contest '%s' is inactive", contestId)
		return nil, ErrContestNotFound
	}

	if contest.State != model.ContestStateRegistrationOpen {
		tracer.Warnf(ctx, "Enrollment failed: contest '%s' state is '%s', not '%s'", contestId, contest.State, model.ContestStateRegistrationOpen)
		return nil, ErrRegistrationClosed
	}

	agent, err := s.repo.GetAgent(ctx, agentId)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Enrollment failed: agent '%s' not found", agentId)
			return nil, ErrAgentNotFound
		}
		return nil, err
	}

	if !agent.Active {
		tracer.Warnf(ctx, "Enrollment failed: agent '%s' is inactive", agentId)
		return nil, ErrAgentNotFound
	}

	// Verify participant ownership unless admin
	if participantId != "" && agent.ParticipantId != participantId {
		isAdmin, permErr := s.repo.HasPermission(ctx, participantId, common.AdminPermission)
		if permErr != nil || !isAdmin {
			tracer.Warnf(ctx, "Enrollment failed: agent '%s' (owner '%s') does not belong to participant '%s'", agentId, agent.ParticipantId, participantId)
			return nil, ErrUnauthorizedAgent
		}
	}

	// Verify game compatibility
	if contest.GameId != "" && agent.GameId != "" && contest.GameId != agent.GameId {
		tracer.Warnf(ctx, "Enrollment failed: agent game '%s' does not match contest game '%s'", agent.GameId, contest.GameId)
		return nil, ErrGameMismatch
	}

	// Verify agent not already enrolled
	existingRanking, err := s.repo.GetRankingByContestAndAgent(ctx, contestId, agentId)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if existingRanking != nil {
		tracer.Warnf(ctx, "Enrollment failed: agent '%s' is already enrolled in contest '%s'", agentId, contestId)
		return nil, ErrAgentAlreadyEnrolled
	}

	ranking := &model.Ranking{
		Id:            uuid.New().String(),
		ContestId:     contestId,
		AgentId:       agentId,
		ParticipantId: agent.ParticipantId,
		Score:         0,
		MatchesPlayed: 0,
		Wins:          0,
		Losses:        0,
		Draws:         0,
		Rank:          1,
		UpdatedAt:     time.Now().UTC(),
	}

	err = s.repo.UpsertRanking(ctx, ranking)
	if err != nil {
		tracer.Errorf(ctx, "Failed to persist enrollment for agent '%s' in contest '%s': %s", agentId, contestId, err)
		return nil, err
	}

	tracer.Debugf(ctx, "Agent '%s' successfully enrolled in contest '%s' with ranking ID '%s'", agentId, contestId, ranking.Id)
	return ranking, nil
}

func (s *service) ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error) {
	tracer.Debugf(ctx, "Listing agents enrolled in contest '%s'", contestId)

	_, err := s.repo.GetContest(ctx, contestId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	rankings, err := s.repo.ListRankingsByContest(ctx, contestId)
	if err != nil {
		return nil, err
	}
	if rankings == nil {
		rankings = make([]model.Ranking, 0)
	}
	return rankings, nil
}

func (s *service) ListCategories(ctx context.Context) ([]model.Category, error) {
	return s.repo.ListCategories(ctx)
}

func (s *service) GetCategory(ctx context.Context, id string) (*model.Category, error) {
	return s.repo.GetCategory(ctx, id)
}

func (s *service) CreateCategory(ctx context.Context, category *model.Category) error {
	if category.Id == "" {
		category.Id = uuid.New().String()
	}
	category.Active = true
	return s.repo.CreateCategory(ctx, category)
}

func (s *service) UpdateCategory(ctx context.Context, category *model.Category) error {
	return s.repo.UpdateCategory(ctx, category)
}

func (s *service) ActivateCategory(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateCategory(ctx, id, isActive)
}
