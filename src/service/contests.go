package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/google/uuid"
)

var (
	ErrInvalidStateFilter   = errors.New("the specified state filter is invalid")
	ErrPrivateStateFilter   = errors.New("the specified state is private and cannot be queried publicly")
	ErrContestNotFound      = errors.New("contest not found")
	ErrRegistrationClosed   = errors.New("contest is not currently open for registration")
	ErrAgentNotFound        = errors.New("agent not found")
	ErrUnauthorizedAgent    = errors.New("agent does not belong to the authenticated user")
	ErrGameMismatch         = errors.New("agent is not configured for the contest's game")
	ErrAgentAlreadyEnrolled = errors.New("agent is already enrolled in this contest")
	ErrUnsupportedGame      = errors.New("Agentrix MVP supports only starfighter")
	ErrNoReadySubmission    = errors.New("agent has no ready submission for contest enrollment")
	ErrSubmissionMismatch   = errors.New("specified submission does not belong to the enrolled agent")
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
	EnrollAgent(ctx context.Context, userId string, contestId string, agentId string, submissionId ...string) (*model.ContestEntry, *model.Ranking, error)
	ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error)
}

func (s *service) ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error) {
	if filter.State != "" {
		if !filter.State.IsValid() {
			return nil, ErrInvalidStateFilter
		}
		if !filter.State.IsPublic() {
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
	if contest.GameId != "starfighter" {
		return ErrUnsupportedGame
	}
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
	if contest.GameId != "starfighter" {
		return ErrUnsupportedGame
	}
	return s.repo.UpdateContest(ctx, contest)
}

func (s *service) ActivateContest(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateContest(ctx, id, isActive)
}

func (s *service) GetPublicContest(ctx context.Context, id string) (*model.Contest, error) {
	c, err := s.repo.GetContest(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	if !c.Active || !c.State.IsPublic() {
		return nil, ErrContestNotFound
	}

	return c, nil
}

func (s *service) EnrollAgent(ctx context.Context, userId string, contestId string, agentId string, submissionId ...string) (*model.ContestEntry, *model.Ranking, error) {
	contest, err := s.repo.GetContest(ctx, contestId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrContestNotFound
		}
		return nil, nil, err
	}

	if !contest.Active {
		return nil, nil, ErrContestNotFound
	}

	if contest.State != model.ContestStateRegistrationOpen {
		return nil, nil, ErrRegistrationClosed
	}

	agent, err := s.repo.GetAgent(ctx, agentId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrAgentNotFound
		}
		return nil, nil, err
	}

	if !agent.Active {
		return nil, nil, ErrAgentNotFound
	}

	ownerID := agent.OwnerUserId

	// Verify user ownership unless admin
	if userId != "" && ownerID != userId {
		isAdmin, permErr := s.repo.HasPermission(ctx, userId, common.AdminPermission)
		if permErr != nil || !isAdmin {
			return nil, nil, ErrUnauthorizedAgent
		}
	}

	// Verify game compatibility
	if contest.GameId != "" && agent.GameId != "" && contest.GameId != agent.GameId {
		return nil, nil, ErrGameMismatch
	}

	// Resolve and lock submission
	var chosenSubmissionId string
	if len(submissionId) > 0 && submissionId[0] != "" {
		sub, err := s.repo.GetSubmission(ctx, submissionId[0])
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, nil, ErrSubmissionNotFound
			}
			return nil, nil, err
		}
		if sub.AgentId != agentId {
			return nil, nil, ErrSubmissionMismatch
		}
		if sub.Status != "ready" {
			return nil, nil, ErrNoReadySubmission
		}
		chosenSubmissionId = sub.Id
	} else {
		subs, err := s.repo.ListSubmissionsByAgent(ctx, agentId)
		if err != nil {
			return nil, nil, err
		}
		for _, sub := range subs {
			if sub.Status == "ready" && sub.Active {
				chosenSubmissionId = sub.Id
				break
			}
		}
		if chosenSubmissionId == "" {
			return nil, nil, ErrNoReadySubmission
		}
	}

	// Verify agent not already enrolled in contest_entries or rankings
	existingEntry, err := s.repo.GetContestEntry(ctx, contestId, agentId)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}
	if existingEntry != nil {
		return nil, nil, ErrAgentAlreadyEnrolled
	}
	existingRanking, err := s.repo.GetRankingByContestAndAgent(ctx, contestId, agentId)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}
	if existingRanking != nil {
		return nil, nil, ErrAgentAlreadyEnrolled
	}

	// 1. Create ContestEntry locking chosenSubmissionId
	entry := &model.ContestEntry{
		Id:           uuid.New().String(),
		ContestId:    contestId,
		AgentId:      agentId,
		UserId:       ownerID,
		SubmissionId: chosenSubmissionId,
		Status:       "enrolled",
		EnrolledAt:   time.Now().UTC(),
		Agent:        agent,
	}
	if err := s.repo.CreateContestEntry(ctx, entry); err != nil {
		return nil, nil, err
	}

	// 2. Create or Upsert initial Ranking row for leaderboard
	ranking := &model.Ranking{
		Id:            uuid.New().String(),
		ContestId:     contestId,
		AgentId:       agentId,
		UserId:        ownerID,
		Score:         0,
		MatchesPlayed: 0,
		Wins:          0,
		Losses:        0,
		Draws:         0,
		Rank:          1,
		UpdatedAt:     time.Now().UTC(),
		Agent:         agent,
	}

	err = s.repo.UpsertRanking(ctx, ranking)
	if err != nil {
		return nil, nil, err
	}

	return entry, ranking, nil
}

func (s *service) ListContestEntries(ctx context.Context, contestId string) ([]model.ContestEntry, error) {
	_, err := s.repo.GetContest(ctx, contestId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrContestNotFound
		}
		return nil, err
	}

	entries, err := s.repo.ListContestEntries(ctx, contestId)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = make([]model.ContestEntry, 0)
	}
	return entries, nil
}

func (s *service) ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error) {
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

func (s *service) UpdateContestStateCAS(ctx context.Context, contestId string, expectedState, newState model.ContestState) (bool, error) {
	if err := expectedState.ValidateTransition(newState); err != nil {
		return false, err
	}
	if casRepo, ok := s.repo.(repository.ContestCASRepository); ok {
		return casRepo.UpdateContestStateCAS(ctx, contestId, expectedState, newState)
	}
	return false, errors.New("contest repository does not support CAS")
}
