package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

type MatchInput struct {
	ContestID string
	Mode      model.MatchMode
	// EntryIDs define the ordered roster of a competitive match.
	EntryIDs []string
	// SubmissionIDs define the ordered roster of exhibition/demo matches.
	SubmissionIDs []string
	Seed          *int64
}

func randomSeed() (int64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(b[:]) & 0x7fffffffffffffff), nil
}

// CreateMatch builds a match and its immutable slots. Competitive rosters
// come exclusively from enrolled contest entries (their locked submission);
// exhibition and demo matches take ready submissions and never affect
// rankings.
func (s *Service) CreateMatch(ctx context.Context, p model.Principal, in MatchInput) (*model.Match, error) {
	if err := requireCap(p, model.CapMatchesCreate); err != nil {
		return nil, err
	}
	if !in.Mode.Valid() {
		return nil, model.Validation("invalid_mode", "mode must be competitive, exhibition or demo")
	}
	seed := int64(0)
	if in.Seed != nil {
		if *in.Seed < 0 {
			return nil, model.Validation("invalid_seed", "seed must be non-negative")
		}
		seed = *in.Seed
	} else {
		var err error
		if seed, err = randomSeed(); err != nil {
			return nil, err
		}
	}
	var matchID string
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		var contest *model.Contest
		if in.ContestID != "" {
			var err error
			if contest, err = q.ShareLockContest(ctx, in.ContestID); err != nil {
				return err
			}
		}
		type slotSource struct {
			agent   *model.Agent
			sub     *model.Submission
			entryID string
			label   string
		}
		var sources []slotSource
		gameID := ""
		switch in.Mode {
		case model.ModeCompetitive:
			if contest == nil {
				return model.Validation("contest_required", "competitive matches require contest_id")
			}
			if len(in.SubmissionIDs) > 0 {
				return model.Validation("entries_required", "competitive rosters are built from entry_ids, not submission_ids")
			}
			if contest.State != model.ContestRegistrationClosed && contest.State != model.ContestRunning {
				return model.Conflict("contest_not_scheduling", "competitive matches can be created once registration is closed (state %s)", contest.State)
			}
			entries, err := q.LockEntriesByIDs(ctx, contest.ID, in.EntryIDs)
			if err != nil {
				return err
			}
			seenAgents := map[string]bool{}
			for _, e := range entries {
				if e.Status != model.EntryEnrolled {
					return model.Validation("entry_not_eligible", "entry %s is %s", e.ID, e.Status)
				}
				if seenAgents[e.AgentID] {
					return model.Validation("duplicate_competitor", "agent %s appears twice in the roster", e.AgentName)
				}
				seenAgents[e.AgentID] = true
				sub, err := q.LockSubmission(ctx, e.SubmissionID)
				if err != nil {
					return err
				}
				agent, err := q.GetAgent(ctx, e.AgentID)
				if err != nil {
					return err
				}
				if sub.Status != model.SubmissionReady {
					return model.Validation("submission_not_ready", "locked submission of %s is %s", e.AgentName, sub.Status)
				}
				sources = append(sources, slotSource{agent: agent, sub: sub, entryID: e.ID,
					label: fmt.Sprintf("%s (%s) v%d", agent.Name, e.Username, sub.Version)})
			}
			gameID = contest.GameID
		default:
			if len(in.EntryIDs) > 0 {
				return model.Validation("submissions_required", "exhibition and demo rosters use submission_ids")
			}
			if contest != nil && !contest.State.Public() {
				return model.Conflict("contest_draft", "matches cannot be attached to a draft contest")
			}
			for _, subID := range in.SubmissionIDs {
				sub, err := q.LockSubmission(ctx, subID)
				if err != nil {
					return err
				}
				if sub.Status != model.SubmissionReady {
					return model.Validation("submission_not_ready", "submission %s is %s", sub.ID, sub.Status)
				}
				agent, err := q.GetAgent(ctx, sub.AgentID)
				if err != nil {
					return err
				}
				if gameID == "" {
					gameID = agent.GameID
				}
				if agent.GameID != gameID {
					return model.Validation("game_mismatch", "all submissions must target the same game")
				}
				sources = append(sources, slotSource{agent: agent, sub: sub,
					label: fmt.Sprintf("%s (%s) v%d", agent.Name, agent.OwnerName, sub.Version)})
			}
			if contest != nil && contest.GameID != gameID {
				return model.Validation("game_mismatch", "submissions do not target the contest game")
			}
		}
		if len(sources) == 0 {
			return model.Validation("empty_roster", "a match needs at least one slot")
		}
		module, err := s.module(gameID)
		if err != nil {
			return err
		}
		if err := module.ValidateRoster(len(sources)); err != nil {
			return err
		}
		match := &model.Match{ID: s.newID(), GameID: gameID, Mode: in.Mode, State: model.MatchScheduled, Seed: seed,
			CreatedBy: p.UserID, CreatedAt: now}
		if contest != nil {
			match.ContestID = contest.ID
		}
		if err := q.CreateMatch(ctx, match); err != nil {
			return err
		}
		for i, src := range sources {
			slot := &model.MatchSlot{ID: s.newID(), MatchID: match.ID, SlotIndex: i, AgentID: src.agent.ID,
				SubmissionID: src.sub.ID, ContestEntryID: src.entryID, DisplayName: truncateText(src.label, 160), CreatedAt: now}
			if err := q.CreateSlot(ctx, match.Mode, slot); err != nil {
				return err
			}
		}
		matchID = match.ID
		return q.Audit(ctx, p.UserID, "match.created", "match", match.ID,
			map[string]any{"mode": in.Mode, "contest_id": in.ContestID, "slots": len(sources)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetMatch(ctx, matchID)
}

// RunTicket is the response of ScheduleRun: the run and job really created
// (or replayed from the idempotency record).
type RunTicket struct {
	MatchID string           `json:"match_id"`
	RunID   string           `json:"run_id"`
	JobID   string           `json:"job_id"`
	Attempt int              `json:"attempt"`
	State   model.MatchState `json:"state"`
}

const opScheduleRun = "matches.schedule_run"

// ScheduleRun is the only way to queue a match execution. In a single
// transaction it locks the match, validates state, mode, contest and roster,
// verifies submission digests, reserves the next attempt number, seals the
// ExecutionSpec, creates the run and the job, moves the match to queued and
// stores the idempotency record.
func (s *Service) ScheduleRun(ctx context.Context, p model.Principal, matchID, idempotencyKey string) (*RunTicket, bool, error) {
	if err := requireCap(p, model.CapMatchesRun); err != nil {
		return nil, false, err
	}
	if idempotencyKey == "" {
		return nil, false, model.BadRequest("idempotency_key_required", "the Idempotency-Key header is required")
	}
	requestHash := model.SHA256Hex([]byte(opScheduleRun + "|" + matchID))
	var ticket *RunTicket
	replayed := false
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		if err := q.LockIdempotencyKey(ctx, p.UserID, opScheduleRun, idempotencyKey, now); err != nil {
			return err
		}
		rec, err := q.FindIdempotency(ctx, p.UserID, opScheduleRun, idempotencyKey, now)
		if err != nil {
			return err
		}
		if rec != nil {
			if rec.RequestHash != requestHash {
				return model.Conflict("idempotency_key_reused", "the Idempotency-Key was already used for a different request")
			}
			ticket = &RunTicket{}
			replayed = true
			return json.Unmarshal(rec.ResponseBody, ticket)
		}

		match, err := q.LockMatch(ctx, matchID)
		if err != nil {
			return err
		}
		if match.State == model.MatchQueued || match.State == model.MatchRunning {
			job, err := q.ActiveJob(ctx, match.ID)
			if err != nil {
				return err
			}
			details := map[string]any{"match_state": match.State}
			if job != nil {
				details["run_id"], details["job_id"] = job.RunID, job.ID
			}
			return model.WithDetails(model.Conflict("run_in_progress", "the match already has a run in progress"), details)
		}
		if !match.State.CanTransitionTo(model.MatchQueued) {
			return model.InvalidTransition("match", match.State, model.MatchQueued)
		}
		spec, err := s.buildSpec(ctx, q, match)
		if err != nil {
			return err
		}
		specJSON, specHash, err := spec.Seal()
		if err != nil {
			return err
		}
		attempt, err := q.MaxRunAttempt(ctx, match.ID)
		if err != nil {
			return err
		}
		run := &model.MatchRun{ID: s.newID(), MatchID: match.ID, Attempt: attempt + 1, ExecutionSpecHash: specHash,
			EngineSHA256: spec.Engine.SHA256, CreatedAt: now}
		if err := q.CreateRun(ctx, run, specJSON); err != nil {
			return err
		}
		job := &model.MatchJob{ID: s.newID(), MatchID: match.ID, RunID: run.ID, EngineSHA256: spec.Engine.SHA256,
			AvailableAt: now, CreatedAt: now}
		if err := q.CreateJob(ctx, job); err != nil {
			return err
		}
		if err := q.TransitionMatch(ctx, match.ID, match.State, model.MatchQueued, now); err != nil {
			return err
		}
		ticket = &RunTicket{MatchID: match.ID, RunID: run.ID, JobID: job.ID, Attempt: run.Attempt, State: model.MatchQueued}
		body, err := json.Marshal(ticket)
		if err != nil {
			return err
		}
		if err := q.SaveIdempotency(ctx, p.UserID, opScheduleRun, idempotencyKey, repository.IdempotencyRecord{
			RequestHash: requestHash, ResponseStatus: 202, ResponseBody: body, ResourceType: "match_run", ResourceID: run.ID,
		}, now, now.Add(s.opts.IdempotencyTTL)); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "match.run_scheduled", "match", match.ID,
			map[string]any{"run_id": run.ID, "attempt": run.Attempt, "spec_hash": specHash}, now)
	})
	if err != nil {
		return nil, false, err
	}
	return ticket, replayed, nil
}

// buildSpec assembles the ExecutionSpec from the locked roster, the game
// module and the engine artifact of a live worker. Every digest is checked.
func (s *Service) buildSpec(ctx context.Context, q *repository.Queries, match *model.Match) (*model.ExecutionSpec, error) {
	module, err := s.module(match.GameID)
	if err != nil {
		return nil, err
	}
	if err := module.ValidateRoster(len(match.Slots)); err != nil {
		return nil, err
	}
	if match.Mode == model.ModeCompetitive {
		contest, err := q.ShareLockContest(ctx, match.ContestID)
		if err != nil {
			return nil, err
		}
		if contest.State != model.ContestRunning {
			return nil, model.Conflict("contest_not_running", "competitive runs require a running contest (state %s)", contest.State)
		}
	}
	engine, err := q.SelectEngineArtifact(ctx, module.ID, module.Version, module.EngineProtocol, s.now().Add(-s.opts.WorkerLiveWindow))
	if err != nil {
		return nil, err
	}
	spec := &model.ExecutionSpec{
		SpecVersion: model.ExecutionSpecVersion, MatchID: match.ID, Mode: match.Mode, Seed: match.Seed,
		TickRate: module.TickRate, Limits: module.ExecutionLimits(),
		Game:         model.SpecGame{ID: module.ID, Version: module.Version, Config: module.ConfigCopy()},
		Engine:       model.SpecEngine{Version: engine.EngineVersion, SHA256: engine.SHA256, ProtocolVersion: engine.ProtocolVersion},
		Runtime:      model.SpecRuntime{BotProtocolVersion: module.BotProtocol, SandboxProfile: "agentrix-sandbox/1"},
		ReplayFormat: module.ReplayFormat,
	}
	for _, slot := range match.Slots {
		sub, err := q.LockSubmission(ctx, slot.SubmissionID)
		if err != nil {
			return nil, err
		}
		if sub.Status != model.SubmissionReady {
			return nil, model.Conflict("submission_not_ready", "submission of slot %d is %s", slot.SlotIndex, sub.Status)
		}
		if match.Mode == model.ModeCompetitive {
			entry, err := q.LockEntry(ctx, match.ContestID, slot.ContestEntryID)
			if err != nil {
				return nil, err
			}
			if entry.Status != model.EntryEnrolled || entry.SubmissionID != slot.SubmissionID {
				return nil, model.Conflict("entry_not_eligible", "entry of slot %d is %s", slot.SlotIndex, entry.Status)
			}
		}
		if !model.IsSHA256(sub.ArtifactSHA256) {
			return nil, model.Conflict("artifact_missing", "submission of slot %d has no digest", slot.SlotIndex)
		}
		if err := s.artifacts.Verify(sub.ArtifactKey, sub.ArtifactSHA256); err != nil {
			return nil, model.Conflict("artifact_corrupt", "artifact of slot %d is missing or corrupt", slot.SlotIndex)
		}
		spec.Slots = append(spec.Slots, model.SpecSlot{Index: slot.SlotIndex, SlotID: slot.ID, ContestEntryID: slot.ContestEntryID,
			AgentID: slot.AgentID, SubmissionID: sub.ID, ArtifactKey: sub.ArtifactKey, ArtifactSHA256: sub.ArtifactSHA256,
			Runtime: sub.Runtime, Entrypoint: sub.Entrypoint})
	}
	return spec, nil
}

// CancelMatch cancels a match that is not executing. A queued match can be
// cancelled only while its job is still pending.
func (s *Service) CancelMatch(ctx context.Context, p model.Principal, matchID, reason string) (*model.Match, error) {
	if err := requireCap(p, model.CapMatchesCancel); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		match, err := q.LockMatch(ctx, matchID)
		if err != nil {
			return err
		}
		if !match.State.CanTransitionTo(model.MatchCancelled) {
			return model.InvalidTransition("match", match.State, model.MatchCancelled)
		}
		if match.State == model.MatchQueued {
			job, err := q.ActiveJob(ctx, matchID)
			if err != nil {
				return err
			}
			if job == nil || job.State != model.JobPending {
				return model.Conflict("run_started", "the run has already been reserved by a worker")
			}
			if err := q.CancelPendingJob(ctx, job.ID, now); err != nil {
				return err
			}
			if err := q.EndRun(ctx, job.RunID, 0, model.RunAborted, model.ErrorCancelled, "match cancelled", now); err != nil {
				return err
			}
		}
		if err := q.TransitionMatch(ctx, matchID, match.State, model.MatchCancelled, now); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "match.cancelled", "match", matchID, map[string]any{"reason": truncateText(reason, 500)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetMatch(ctx, matchID)
}

// MatchDetail is the read model of a match with its runs and the official
// results/replay of the committed run.
type MatchDetail struct {
	Match   *model.Match
	Runs    []model.MatchRun
	Results []model.Result
	Replay  *model.Replay
}

func (s *Service) visibleMatch(ctx context.Context, p model.Principal, id string) (*model.Match, error) {
	if err := requireCap(p, model.CapMatchesView); err != nil {
		return nil, err
	}
	match, err := s.store.GetMatch(ctx, id)
	if err != nil {
		return nil, err
	}
	if match.ContestID != "" && !p.Can(model.CapContestsManage) {
		contest, err := s.store.GetContest(ctx, match.ContestID)
		if err != nil {
			return nil, err
		}
		if !contest.State.Public() {
			return nil, model.NotFound("match_not_found", "match not found")
		}
	}
	return match, nil
}

func (s *Service) GetMatchDetail(ctx context.Context, p model.Principal, id string) (*MatchDetail, error) {
	match, err := s.visibleMatch(ctx, p, id)
	if err != nil {
		return nil, err
	}
	detail := &MatchDetail{Match: match}
	if detail.Runs, err = s.store.ListRuns(ctx, id); err != nil {
		return nil, err
	}
	if match.CommittedRunID != "" {
		if detail.Results, err = s.store.ListResultsByRun(ctx, match.CommittedRunID); err != nil {
			return nil, err
		}
		if detail.Replay, err = s.store.GetReplayByRun(ctx, match.CommittedRunID); err != nil {
			return nil, err
		}
	}
	return detail, nil
}

func (s *Service) ListMatches(ctx context.Context, p model.Principal, contestID string) ([]model.Match, error) {
	if err := requireCap(p, model.CapMatchesView); err != nil {
		return nil, err
	}
	if contestID != "" {
		if _, err := s.visibleContest(ctx, p, contestID); err != nil {
			return nil, err
		}
	}
	return s.store.ListMatches(ctx, repository.MatchFilter{ContestID: contestID, PublicOnly: !p.Can(model.CapContestsManage)})
}
