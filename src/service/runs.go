package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

// RegisterWorker announces a worker and the engine artifact it runs. The
// artifact must match an installed game module.
func (s *Service) RegisterWorker(ctx context.Context, info model.WorkerInfo, artifact model.EngineArtifact) error {
	module, err := s.module(artifact.GameID)
	if err != nil {
		return err
	}
	if artifact.GameVersion != module.Version || artifact.ProtocolVersion != module.EngineProtocol {
		return fmt.Errorf("engine artifact %s targets %s/%s, module declares %s/%s", artifact.SHA256,
			artifact.GameVersion, artifact.ProtocolVersion, module.Version, module.EngineProtocol)
	}
	if !model.IsSHA256(artifact.SHA256) || artifact.EngineVersion == "" {
		return fmt.Errorf("engine artifact requires sha256 and version")
	}
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		if err := q.UpsertGame(ctx, model.Game{ID: module.ID, Name: module.Name, Version: module.Version,
			MinPlayers: module.Players.Min, MaxPlayers: module.Players.Max}, now); err != nil {
			return err
		}
		return q.RegisterWorker(ctx, info, artifact, now)
	})
}

// SyncGames registers every loaded module in the games table.
func (s *Service) SyncGames(ctx context.Context) error {
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		for _, m := range s.games.List() {
			if err := q.UpsertGame(ctx, model.Game{ID: m.ID, Name: m.Name, Version: m.Version,
				MinPlayers: m.Players.Min, MaxPlayers: m.Players.Max}, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) TouchWorker(ctx context.Context, workerID string) error {
	return s.store.TouchWorker(ctx, workerID, s.now())
}

// ReserveNext reserves the oldest pending job for the worker's engine. The
// job, its run and its match move together: pending/created/queued to
// reserved/reserved/running, under a fresh fencing token.
func (s *Service) ReserveNext(ctx context.Context, workerID, engineSHA, sandbox string) (*model.Reservation, error) {
	var res *model.Reservation
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		job, err := q.LockNextPendingJob(ctx, engineSHA, now)
		if err != nil || job == nil {
			return err
		}
		token, err := q.NextFencingToken(ctx)
		if err != nil {
			return err
		}
		lease := now.Add(s.opts.LeaseTTL)
		if err := q.ReserveJob(ctx, job.ID, workerID, token, lease, now); err != nil {
			return err
		}
		if err := q.ReserveRun(ctx, job.RunID, workerID, sandbox, token, now); err != nil {
			return err
		}
		match, err := q.LockMatch(ctx, job.MatchID)
		if err != nil {
			return err
		}
		if err := q.TransitionMatch(ctx, match.ID, model.MatchQueued, model.MatchRunning, now); err != nil {
			return err
		}
		run, err := q.GetRun(ctx, job.RunID)
		if err != nil {
			return err
		}
		res = &model.Reservation{JobID: job.ID, RunID: run.ID, MatchID: match.ID, ContestID: match.ContestID,
			Attempt: run.Attempt, WorkerID: workerID, FencingToken: token, LeaseUntil: lease, Spec: run.ExecutionSpec}
		return q.Audit(ctx, "", "run.reserved", "match_run", run.ID, map[string]any{"worker_id": workerID, "fencing_token": token}, now)
	})
	return res, err
}

// StartRun marks the reserved run as executing.
func (s *Service) StartRun(ctx context.Context, r model.Reservation) error {
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		if _, err := q.LockReservedJob(ctx, r, now); err != nil {
			return err
		}
		return q.StartRun(ctx, r.RunID, r.FencingToken, now)
	})
}

// Heartbeat renews the lease; any error means the reservation is lost and
// the executor must stop.
func (s *Service) Heartbeat(ctx context.Context, r *model.Reservation) error {
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		lease := now.Add(s.opts.LeaseTTL)
		if err := q.RenewLease(ctx, r.JobID, r.WorkerID, r.FencingToken, lease, now); err != nil {
			return err
		}
		if err := q.HeartbeatRun(ctx, r.RunID, r.FencingToken, now); err != nil {
			return err
		}
		r.LeaseUntil = lease
		return nil
	})
}

// validateRanks accepts standard competition rankings ("1224"): the rank of
// each slot equals one plus the number of slots with a strictly better rank.
func validateRanks(ranks []int) error {
	sorted := append([]int(nil), ranks...)
	sort.Ints(sorted)
	for i, r := range sorted {
		if r < 1 {
			return fmt.Errorf("rank %d is not positive", r)
		}
		first := sort.SearchInts(sorted, r)
		if r != first+1 {
			return fmt.Errorf("ranks %v are not a valid competition ranking", ranks)
		}
		_ = i
	}
	return nil
}

func outcomes(slots []model.SlotOutcome) map[string]model.Outcome {
	out := map[string]model.Outcome{}
	topRank := 0
	for _, sl := range slots {
		if !sl.Disqualified && (topRank == 0 || sl.Rank < topRank) {
			topRank = sl.Rank
		}
	}
	winners := 0
	for _, sl := range slots {
		if !sl.Disqualified && sl.Rank == topRank {
			winners++
		}
	}
	for _, sl := range slots {
		switch {
		case sl.Disqualified:
			out[sl.SlotID] = model.OutcomeDisqualified
		case sl.Rank == topRank && winners == 1:
			out[sl.SlotID] = model.OutcomeWin
		case sl.Rank == topRank:
			out[sl.SlotID] = model.OutcomeDraw
		default:
			out[sl.SlotID] = model.OutcomeLoss
		}
	}
	return out
}

func replayKeys(runID, compression string) (staging, storage string) {
	ext := "ndjson"
	if compression == "gzip" {
		ext = "ndjson.gz"
	}
	return "replays/staging/" + runID + "." + ext, "replays/published/" + runID + "." + ext
}

// ReplayKeys exposes the canonical storage keys of a run's replay.
func ReplayKeys(runID, compression string) (string, string) { return replayKeys(runID, compression) }

// CommitRun is the single owner of a successful run's terminal transition.
// In one transaction it verifies the reservation (owner, positive exact
// fencing token, live lease, run identity), validates the reported results
// against the match slots, writes results and replay metadata, and moves
// job, run and match to their terminal states.
func (s *Service) CommitRun(ctx context.Context, c model.RunCompletion) error {
	r := c.Reservation
	if c.Replay.FormatVersion != r.Spec.ReplayFormat {
		return model.Validation("replay_format_mismatch", "replay format %q does not match the spec", c.Replay.FormatVersion)
	}
	staging, storage := replayKeys(r.RunID, c.Replay.Compression)
	if c.Replay.StagingKey != staging || c.Replay.StorageKey != storage || c.Replay.SizeBytes <= 0 ||
		c.Replay.FrameCount <= 0 || !model.IsSHA256(c.Replay.SHA256) {
		return model.Validation("invalid_replay", "replay metadata does not belong to run %s", r.RunID)
	}
	if err := s.artifacts.Verify(staging, c.Replay.SHA256); err != nil {
		return model.Validation("invalid_replay", "staged replay cannot be verified: %v", err)
	}
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		if _, err := q.LockReservedJob(ctx, r, now); err != nil {
			return err
		}
		run, err := q.LockRun(ctx, r.RunID)
		if err != nil {
			return err
		}
		if run.MatchID != r.MatchID || run.State != model.RunRunning || run.FencingToken != r.FencingToken {
			return repository.ErrFenced
		}
		match, err := q.LockMatch(ctx, r.MatchID)
		if err != nil {
			return err
		}
		if match.State != model.MatchRunning {
			return model.InvalidTransition("match", match.State, model.MatchFinished)
		}
		bySlot := map[string]model.SlotOutcome{}
		ranks := make([]int, 0, len(c.Slots))
		for _, sl := range c.Slots {
			if _, dup := bySlot[sl.SlotID]; dup {
				return model.Validation("duplicate_result", "slot %s reported twice", sl.SlotID)
			}
			bySlot[sl.SlotID] = sl
			ranks = append(ranks, sl.Rank)
		}
		if len(bySlot) != len(match.Slots) {
			return model.Validation("result_count_mismatch", "expected %d results, got %d", len(match.Slots), len(bySlot))
		}
		for _, slot := range match.Slots {
			if _, ok := bySlot[slot.ID]; !ok {
				return model.Validation("foreign_or_missing_slot", "no result for slot %d of this match", slot.SlotIndex)
			}
		}
		if err := validateRanks(ranks); err != nil {
			return model.Validation("invalid_ranks", "%v", err)
		}
		outcome := outcomes(c.Slots)
		for _, slot := range match.Slots {
			sl := bySlot[slot.ID]
			details, err := json.Marshal(sl.Details)
			if err != nil {
				return err
			}
			if sl.Details == nil {
				details = []byte(`{}`)
			}
			if err := q.InsertResult(ctx, &model.Result{ID: s.newID(), MatchRunID: run.ID, MatchID: match.ID, SlotID: slot.ID,
				SubmissionID: slot.SubmissionID, Score: sl.Score, Rank: sl.Rank, Outcome: outcome[slot.ID], Details: details,
				CreatedAt: now}); err != nil {
				return err
			}
		}
		if err := q.InsertReplay(ctx, &model.Replay{ID: c.Replay.ID, MatchRunID: run.ID, MatchID: match.ID,
			FormatVersion: c.Replay.FormatVersion, Compression: c.Replay.Compression, StagingKey: staging, StorageKey: storage,
			SHA256: c.Replay.SHA256, SizeBytes: c.Replay.SizeBytes, FrameCount: c.Replay.FrameCount, CreatedAt: now}); err != nil {
			return err
		}
		if err := q.EndRun(ctx, run.ID, r.FencingToken, model.RunCommitted, "", "", now); err != nil {
			return err
		}
		if err := q.EndJob(ctx, r.JobID, r.FencingToken, model.JobCompleted, "", now); err != nil {
			return err
		}
		if err := q.FinishMatch(ctx, match.ID, run.ID, now); err != nil {
			return err
		}
		if match.Mode == model.ModeCompetitive && match.ContestID != "" {
			if err := q.BumpRankingDirty(ctx, match.ContestID); err != nil {
				return err
			}
		}
		return q.Audit(ctx, "", "run.committed", "match_run", run.ID, map[string]any{
			"worker_id": r.WorkerID, "fencing_token": r.FencingToken, "final_tick": c.FinalTick,
			"final_state_hash": c.FinalStateHash, "reason": c.TerminationReason}, now)
	})
}

// FailRun ends a run the worker could not complete. Retryable failures
// schedule a new run (new attempt, same sealed spec) in the same
// transaction; a failed run is never revived.
func (s *Service) FailRun(ctx context.Context, r model.Reservation, class model.ErrorClass, message string) error {
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		if _, err := q.LockReservedJob(ctx, r, now); err != nil {
			return err
		}
		to := model.RunFailed
		if class == model.ErrorLeaseLost {
			to = model.RunTimedOut
		}
		return s.failLocked(ctx, q, r.JobID, r.RunID, r.MatchID, r.Attempt, r.FencingToken, to, class, message, now)
	})
}

func (s *Service) failLocked(ctx context.Context, q *repository.Queries, jobID, runID, matchID string, attempt int, token int64,
	to model.RunState, class model.ErrorClass, message string, now time.Time) error {
	message = sanitizeError(message)
	if err := q.EndRun(ctx, runID, token, to, class, message, now); err != nil {
		return err
	}
	if err := q.EndJob(ctx, jobID, token, model.JobFailed, message, now); err != nil {
		return err
	}
	match, err := q.LockMatch(ctx, matchID)
	if err != nil {
		return err
	}
	if err := q.TransitionMatch(ctx, matchID, match.State, model.MatchFailed, now); err != nil {
		return err
	}
	retry := class.Retryable() && attempt < s.opts.MaxRunAttempts
	if err := q.Audit(ctx, "", "run.failed", "match_run", runID,
		map[string]any{"class": class, "state": to, "retry": retry}, now); err != nil {
		return err
	}
	if !retry {
		return nil
	}
	prev, err := q.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	specJSON, err := q.SpecJSON(ctx, runID)
	if err != nil {
		return err
	}
	next := &model.MatchRun{ID: s.newID(), MatchID: matchID, Attempt: attempt + 1, ExecutionSpecHash: prev.ExecutionSpecHash,
		EngineSHA256: prev.EngineSHA256, CreatedAt: now}
	if err := q.CreateRun(ctx, next, specJSON); err != nil {
		return err
	}
	backoff := time.Duration(attempt*attempt) * 2 * time.Second
	if err := q.CreateJob(ctx, &model.MatchJob{ID: s.newID(), MatchID: matchID, RunID: next.ID, EngineSHA256: prev.EngineSHA256,
		AvailableAt: now.Add(backoff), CreatedAt: now}); err != nil {
		return err
	}
	if err := q.TransitionMatch(ctx, matchID, model.MatchFailed, model.MatchQueued, now); err != nil {
		return err
	}
	return q.Audit(ctx, "", "run.retry_scheduled", "match_run", next.ID, map[string]any{"previous_run_id": runID, "attempt": next.Attempt}, now)
}

// ReapExpiredLeases times out runs whose worker stopped renewing its lease
// (crash, partition) and schedules the retry. A worker that comes back
// later is rejected by fencing.
func (s *Service) ReapExpiredLeases(ctx context.Context, limit int) (int, error) {
	reaped := 0
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		jobs, err := q.LockExpiredLeases(ctx, now, limit)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			run, err := q.LockRun(ctx, job.RunID)
			if err != nil {
				return err
			}
			if err := s.failLocked(ctx, q, job.ID, run.ID, job.MatchID, run.Attempt, job.FencingToken,
				model.RunTimedOut, model.ErrorLeaseLost, "worker lease expired", now); err != nil {
				return err
			}
			reaped++
		}
		return nil
	})
	return reaped, err
}

// sanitizeError keeps failure messages short and free of host paths.
func sanitizeError(message string) string {
	message = strings.TrimSpace(message)
	fields := strings.Fields(message)
	for i, f := range fields {
		if strings.HasPrefix(f, "/") && strings.Count(f, "/") > 1 {
			fields[i] = "<path>"
		}
	}
	return truncateText(strings.Join(fields, " "), 1000)
}
