package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/model"
	replaystream "github.com/Agentrix-Group/Agentrix/src/replay"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

type Service interface {
	// Auth & Users
	Login(ctx context.Context, username, password string) (*model.User, error)
	Register(ctx context.Context, user *model.User) error
	ListUsers(ctx context.Context) ([]model.User, error)
	GetUser(ctx context.Context, id string) (*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	ActivateUser(ctx context.Context, id string, isActive bool) error
	HasPermission(ctx context.Context, userId, permission string) (bool, error)

	// Contests & Categories
	ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	GetPublicContest(ctx context.Context, id string) (*model.Contest, error)
	ListContests(ctx context.Context) ([]model.Contest, error)
	GetContest(ctx context.Context, id string) (*model.Contest, error)
	CreateContest(ctx context.Context, contest *model.Contest) error
	UpdateContest(ctx context.Context, contest *model.Contest) error
	ActivateContest(ctx context.Context, id string, isActive bool) error
	EnrollAgent(ctx context.Context, userId string, contestId string, agentId string) (*model.ContestEntry, *model.Ranking, error)
	ListContestEntries(ctx context.Context, contestId string) ([]model.ContestEntry, error)
	ListContestAgents(ctx context.Context, contestId string) ([]model.Ranking, error)
	ListCategories(ctx context.Context) ([]model.Category, error)
	GetCategory(ctx context.Context, id string) (*model.Category, error)
	CreateCategory(ctx context.Context, category *model.Category) error
	UpdateCategory(ctx context.Context, category *model.Category) error
	ActivateCategory(ctx context.Context, id string, isActive bool) error

	// Games
	ListGames(ctx context.Context) ([]model.Game, error)
	GetGame(ctx context.Context, id string) (*model.Game, error)

	// Agents
	ListAgents(ctx context.Context) ([]model.Agent, error)
	ListAgentsByOwner(ctx context.Context, ownerUserId string) ([]model.Agent, error)
	GetAgent(ctx context.Context, id string) (*model.Agent, error)
	CreateAgent(ctx context.Context, agent *model.Agent) error
	UpdateAgent(ctx context.Context, agent *model.Agent) error
	ActivateAgent(ctx context.Context, id string, isActive bool) error

	// Submissions
	ListSubmissions(ctx context.Context) ([]model.Submission, error)
	ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error)
	GetSubmission(ctx context.Context, id string) (*model.Submission, error)
	CreateSubmissionBundle(ctx context.Context, userId, roleId, agentId string, archive []byte) (*model.Submission, error)

	// Matches
	ListMatches(ctx context.Context) ([]model.Match, error)
	ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error)
	GetMatch(ctx context.Context, id string) (*model.Match, error)
	CreateMatch(ctx context.Context, match *model.Match, submissionIds []string) error
	RunMatch(ctx context.Context, matchId string) error
	UpdateMatch(ctx context.Context, match *model.Match) error
	ActivateMatch(ctx context.Context, id string, isActive bool) error

	// Results
	ListResults(ctx context.Context) ([]model.Result, error)
	ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error)
	GetResult(ctx context.Context, id string) (*model.Result, error)
	CreateResult(ctx context.Context, result *model.Result) error

	// Rankings
	ListRankings(ctx context.Context) ([]model.Ranking, error)
	ListRankingsByContest(ctx context.Context, contestId string) ([]model.Ranking, error)
	GetRanking(ctx context.Context, id string) (*model.Ranking, error)
	CalculateRankings(ctx context.Context, contestId string) ([]model.Ranking, error)

	// Match execution runs & atomic commits
	CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error
	CreateMatchRun(ctx context.Context, run *model.MatchRun) error
	GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error)

	// Replays
	GetReplay(ctx context.Context, id string) (*model.Replay, error)
	OpenReplay(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error)
	StreamReplay(ctx context.Context, id string) ([]byte, error)
	PublishReplay(ctx context.Context, replayID string) (*model.Replay, error)
	DiscardReplay(ctx context.Context, replayID string) error

	// Readiness
	CheckReadiness(ctx context.Context) (map[string]any, error)
}

type service struct {
	repo      repository.Repository
	artifacts connection.ArtifactStore
	queue     connection.JobQueue
	validator BotAdmissionValidator
}

type BotAdmissionValidator interface {
	ValidateBot(ctx context.Context, codePath string) error
}

func NewService(repo repository.Repository, artifacts connection.ArtifactStore, queue connection.JobQueue, validators ...BotAdmissionValidator) Service {
	svc := &service{
		repo:      repo,
		artifacts: artifacts,
		queue:     queue,
	}
	if len(validators) > 0 {
		svc.validator = validators[0]
	}
	return svc
}

func (s *service) CheckReadiness(ctx context.Context) (map[string]any, error) {
	checks := make(map[string]any)

	// 1. Database Ping
	if s.repo == nil {
		checks["database"] = "DOWN: repository is nil"
		return checks, errors.New("repository is nil")
	}
	if err := s.repo.Ping(ctx); err != nil {
		checks["database"] = "DOWN: " + err.Error()
		return checks, fmt.Errorf("database unavailable: %w", err)
	}
	checks["database"] = "UP"

	// 2. Schema compatibility
	if err := s.repo.CheckSchema(ctx); err != nil {
		checks["schema"] = "DOWN: " + err.Error()
		return checks, fmt.Errorf("schema incompatible: %w", err)
	}
	checks["schema"] = "UP"

	// 3. Artifact Store
	if s.artifacts != nil {
		testFile := ".health_check"
		if _, err := s.artifacts.Save(ctx, testFile, []byte("ok")); err != nil {
			checks["artifacts"] = "DOWN: " + err.Error()
			return checks, fmt.Errorf("artifacts store not writable: %w", err)
		}
		_ = s.artifacts.Delete(ctx, testFile)
		checks["artifacts"] = "UP"
	} else {
		checks["artifacts"] = "DEGRADED"
	}

	// 4. Starfighter game registered in DB
	g, err := s.repo.GetGame(ctx, "starfighter")
	if err != nil || g == nil || !g.Active {
		checks["starfighter"] = "DOWN: game starfighter not registered or inactive in database"
		return checks, errors.New("required game 'starfighter' is not active in database")
	}
	checks["starfighter"] = "UP"

	return checks, nil
}
