package service

import (
	"context"

	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/model"
	replaystream "github.com/F4nk1/Agentrix/src/replay"
	"github.com/F4nk1/Agentrix/src/repository"
)

type Service interface {
	// Auth & Participants
	Login(ctx context.Context, username, password string) (*model.Participant, error)
	Register(ctx context.Context, participant *model.Participant) error
	ListParticipants(ctx context.Context) ([]model.Participant, error)
	GetParticipant(ctx context.Context, id string) (*model.Participant, error)
	UpdateParticipant(ctx context.Context, participant *model.Participant) error
	ActivateParticipant(ctx context.Context, id string, isActive bool) error
	HasPermission(ctx context.Context, participantId, permission string) (bool, error)

	// Contests & Categories
	ListPublicContests(ctx context.Context, filter model.PublicContestsFilter) ([]model.PublicContestSummary, error)
	GetPublicContest(ctx context.Context, id string) (*model.Contest, error)
	ListContests(ctx context.Context) ([]model.Contest, error)
	GetContest(ctx context.Context, id string) (*model.Contest, error)
	CreateContest(ctx context.Context, contest *model.Contest) error
	UpdateContest(ctx context.Context, contest *model.Contest) error
	ActivateContest(ctx context.Context, id string, isActive bool) error
	EnrollAgent(ctx context.Context, participantId string, contestId string, agentId string) (*model.Ranking, error)
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
	ListAgentsByParticipant(ctx context.Context, participantId string) ([]model.Agent, error)
	GetAgent(ctx context.Context, id string) (*model.Agent, error)
	CreateAgent(ctx context.Context, agent *model.Agent) error
	UpdateAgent(ctx context.Context, agent *model.Agent) error
	ActivateAgent(ctx context.Context, id string, isActive bool) error

	// Submissions
	ListSubmissions(ctx context.Context) ([]model.Submission, error)
	ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error)
	GetSubmission(ctx context.Context, id string) (*model.Submission, error)
	CreateSubmissionBundle(ctx context.Context, participantId, roleId, agentId string, archive []byte) (*model.Submission, error)

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

	// Replays
	GetReplay(ctx context.Context, id string) (*model.Replay, error)
	OpenReplay(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error)
	StreamReplay(ctx context.Context, id string) ([]byte, error)
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
