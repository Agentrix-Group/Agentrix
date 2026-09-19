package service

import (
	"context"
	"time"

	"github.com/F4nk1/Agentrix/src/common"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

func (s *service) ListMatches(ctx context.Context) ([]model.Match, error) {
	return s.repo.ListMatches(ctx)
}

func (s *service) ListMatchesByContest(ctx context.Context, contestId string) ([]model.Match, error) {
	return s.repo.ListMatchesByContest(ctx, contestId)
}

func (s *service) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	match, err := s.repo.GetMatch(ctx, id)
	if err != nil {
		return nil, err
	}

	// Fetch match results if available
	results, err := s.repo.ListResultsByMatch(ctx, id)
	if err == nil {
		match.Results = results
	}

	return match, nil
}

func (s *service) CreateMatch(ctx context.Context, match *model.Match, submissionIds []string) error {
	if match.GameId != "starfighter" {
		return ErrUnsupportedGame
	}
	if match.Id == "" {
		match.Id = uuid.New().String()
	}
	if match.Status == "" {
		match.Status = common.MatchStatusPending
	}
	if match.Seed == 0 {
		match.Seed = time.Now().UnixNano()
	}
	match.Active = true
	match.CreatedAt = time.Now().UTC()

	if err := s.repo.CreateMatch(ctx, match); err != nil {
		return err
	}

	// Queue job for executor
	if s.queue != nil {
		job := &connection.MatchJob{
			JobId:         uuid.New().String(),
			Attempt:       0,
			MatchId:       match.Id,
			ContestId:     match.ContestId,
			GameId:        match.GameId,
			SubmissionIds: submissionIds,
			Seed:          match.Seed,
		}
		if err := s.queue.Enqueue(ctx, job); err != nil {
			tracer.WarnEvent(ctx, tracer.ScopeQueue, "match.enqueue.degraded", "La partida fue creada, pero no pudo encolarse",
				tracer.Origin(tracer.OriginInfrastructure), tracer.String("job_id", job.JobId),
				tracer.String("match_id", job.MatchId), tracer.Err(err))
		} else {
			tracer.InfoEvent(ctx, tracer.ScopeQueue, "match.queued", "Partida en cola",
				tracer.String("job_id", job.JobId), tracer.String("match_id", job.MatchId))
		}
	}

	return nil
}

func (s *service) RunMatch(ctx context.Context, matchId string) error {
	match, err := s.repo.GetMatch(ctx, matchId)
	if err != nil {
		return err
	}
	if match.GameId != "starfighter" {
		return ErrUnsupportedGame
	}

	// Fetch submissions associated with this match or contest
	var submissionIds []string
	results, _ := s.repo.ListResultsByMatch(ctx, matchId)
	for _, res := range results {
		submissionIds = append(submissionIds, res.SubmissionId)
	}

	if s.queue != nil {
		job := &connection.MatchJob{
			JobId:         uuid.New().String(),
			Attempt:       0,
			MatchId:       match.Id,
			ContestId:     match.ContestId,
			GameId:        match.GameId,
			SubmissionIds: submissionIds,
			Seed:          match.Seed,
		}
		if err := s.queue.Enqueue(ctx, job); err != nil {
			return err
		}
		tracer.InfoEvent(ctx, tracer.ScopeQueue, "match.queued", "Partida en cola",
			tracer.String("job_id", job.JobId), tracer.String("match_id", job.MatchId))
		return nil
	}

	return nil
}

func (s *service) UpdateMatch(ctx context.Context, match *model.Match) error {
	if match.GameId != "starfighter" {
		return ErrUnsupportedGame
	}
	return s.repo.UpdateMatch(ctx, match)
}

func (s *service) ActivateMatch(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateMatch(ctx, id, isActive)
}
