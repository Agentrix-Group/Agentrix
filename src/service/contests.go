package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

type ContestInput struct {
	GameID        string
	Name          string
	Description   string
	StartsAt      *time.Time
	EndsAt        *time.Time
	ScoringPolicy *model.ScoringPolicy
}

func validateContest(c *model.Contest) error {
	if n := utf8.RuneCountInString(c.Name); n < 1 || n > 128 {
		return model.Validation("invalid_contest_name", "contest name must contain 1 to 128 characters")
	}
	if utf8.RuneCountInString(c.Description) > 4000 {
		return model.Validation("invalid_contest_description", "description must not exceed 4000 characters")
	}
	if c.StartsAt != nil && c.EndsAt != nil && !c.StartsAt.Before(*c.EndsAt) {
		return model.Validation("invalid_contest_window", "starts_at must be before ends_at")
	}
	return c.ScoringPolicy.Validate()
}

func (s *Service) CreateContest(ctx context.Context, p model.Principal, in ContestInput) (*model.Contest, error) {
	if err := requireCap(p, model.CapContestsManage); err != nil {
		return nil, err
	}
	if _, err := s.module(in.GameID); err != nil {
		return nil, err
	}
	now := s.now()
	policy := model.DefaultScoringPolicy()
	if in.ScoringPolicy != nil {
		policy = *in.ScoringPolicy
	}
	c := &model.Contest{ID: s.newID(), GameID: in.GameID, Name: strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description), State: model.ContestDraft, StartsAt: in.StartsAt, EndsAt: in.EndsAt,
		ScoringPolicy: policy, CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now}
	if err := validateContest(c); err != nil {
		return nil, err
	}
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		if err := q.CreateContest(ctx, c); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "contest.created", "contest", c.ID, nil, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetContest(ctx, c.ID)
}

type ContestPatch struct {
	Name          *string
	Description   *string
	StartsAt      **time.Time
	EndsAt        **time.Time
	ScoringPolicy *model.ScoringPolicy
}

// UpdateContest edits descriptive fields. The scoring policy can only change
// while the contest is a draft (the database enforces the same rule).
func (s *Service) UpdateContest(ctx context.Context, p model.Principal, id string, patch ContestPatch) (*model.Contest, error) {
	if err := requireCap(p, model.CapContestsManage); err != nil {
		return nil, err
	}
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		c, err := q.LockContest(ctx, id)
		if err != nil {
			return err
		}
		if c.State == model.ContestArchived || c.State == model.ContestCancelled || c.State == model.ContestFinished {
			return model.Conflict("contest_closed", "a %s contest cannot be edited", c.State)
		}
		if patch.Name != nil {
			c.Name = strings.TrimSpace(*patch.Name)
		}
		if patch.Description != nil {
			c.Description = strings.TrimSpace(*patch.Description)
		}
		if patch.StartsAt != nil {
			c.StartsAt = *patch.StartsAt
		}
		if patch.EndsAt != nil {
			c.EndsAt = *patch.EndsAt
		}
		if patch.ScoringPolicy != nil {
			if c.State != model.ContestDraft {
				return model.Conflict("scoring_policy_frozen", "the scoring policy is frozen once registration opens")
			}
			c.ScoringPolicy = *patch.ScoringPolicy
		}
		if err := validateContest(c); err != nil {
			return err
		}
		now := s.now()
		if err := q.UpdateContestDetails(ctx, c, now); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "contest.updated", "contest", id, nil, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetContest(ctx, id)
}

// TransitionContest is the only way to change a contest state.
func (s *Service) TransitionContest(ctx context.Context, p model.Principal, id string, to model.ContestState, reason string) (*model.Contest, error) {
	if err := requireCap(p, model.CapContestsManage); err != nil {
		return nil, err
	}
	if !to.Valid() {
		return nil, model.Validation("invalid_state", "unknown contest state %q", to)
	}
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		c, err := q.LockContest(ctx, id)
		if err != nil {
			return err
		}
		if !c.State.CanTransitionTo(to) {
			return model.InvalidTransition("contest", c.State, to)
		}
		if to == model.ContestRegistrationClosed || to == model.ContestRunning {
			entries, err := q.ListEntries(ctx, id)
			if err != nil {
				return err
			}
			enrolled := 0
			for _, e := range entries {
				if e.Status == model.EntryEnrolled {
					enrolled++
				}
			}
			module, err := s.module(c.GameID)
			if err != nil {
				return err
			}
			if to == model.ContestRunning && enrolled < module.Players.Min {
				return model.Validation("not_enough_entries", "a running contest needs at least %d enrolled entries", module.Players.Min)
			}
		}
		now := s.now()
		if err := q.TransitionContest(ctx, id, c.State, to, now); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "contest.transition", "contest", id,
			map[string]any{"from": c.State, "to": to, "reason": truncateText(reason, 500)}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetContest(ctx, id)
}

// visibleContest applies the read policy: drafts are visible only with
// contests:manage.
func (s *Service) visibleContest(ctx context.Context, p model.Principal, id string) (*model.Contest, error) {
	if err := requireCap(p, model.CapContestsView); err != nil {
		return nil, err
	}
	c, err := s.store.GetContest(ctx, id)
	if err != nil {
		return nil, err
	}
	if !c.State.Public() && !p.Can(model.CapContestsManage) {
		return nil, model.NotFound("contest_not_found", "contest not found")
	}
	return c, nil
}

func (s *Service) GetContest(ctx context.Context, p model.Principal, id string) (*model.Contest, error) {
	return s.visibleContest(ctx, p, id)
}

func (s *Service) ListContests(ctx context.Context, p model.Principal) ([]model.Contest, error) {
	if err := requireCap(p, model.CapContestsView); err != nil {
		return nil, err
	}
	return s.store.ListContests(ctx, p.Can(model.CapContestsManage))
}

func (s *Service) ListEntries(ctx context.Context, p model.Principal, contestID string) ([]model.ContestEntry, error) {
	if _, err := s.visibleContest(ctx, p, contestID); err != nil {
		return nil, err
	}
	return s.store.ListEntries(ctx, contestID)
}

// requireEnrollableSubmission checks that a submission can be locked into a
// contest entry of the agent.
func requireEnrollableSubmission(sub *model.Submission, agentID string) error {
	if sub.AgentID != agentID {
		return model.Validation("submission_agent_mismatch", "submission does not belong to the agent")
	}
	if sub.Status != model.SubmissionReady {
		return model.Validation("submission_not_ready", "submission is %s; only ready submissions can be enrolled", sub.Status)
	}
	return nil
}

// Enroll creates a contest entry locked to an exact, ready submission.
// The owner enrolls with entries:create:own; entries:manage:any may enroll
// agents of other users (the entry still belongs to the agent owner).
func (s *Service) Enroll(ctx context.Context, p model.Principal, contestID, agentID, submissionID string) (*model.ContestEntry, error) {
	if agentID == "" || submissionID == "" {
		return nil, model.Validation("missing_fields", "agent_id and submission_id are required")
	}
	var entryID string
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		contest, err := q.ShareLockContest(ctx, contestID)
		if err != nil {
			return err
		}
		if !contest.State.Public() && !p.Can(model.CapContestsManage) {
			return model.NotFound("contest_not_found", "contest not found")
		}
		if contest.State != model.ContestRegistrationOpen {
			return model.Conflict("registration_not_open", "contest registration is not open (state %s)", contest.State)
		}
		agent, err := q.LockAgent(ctx, agentID)
		if err != nil {
			return err
		}
		if !p.CanFor(agent.OwnerUserID, model.CapEntriesCreateOwn, model.CapEntriesManageAny) {
			return model.NotFound("agent_not_found", "agent not found")
		}
		if agent.Status != model.AgentActive {
			return model.Conflict("agent_disabled", "agent is disabled")
		}
		if agent.GameID != contest.GameID {
			return model.Validation("game_mismatch", "agent plays %s but the contest is %s", agent.GameID, contest.GameID)
		}
		sub, err := q.LockSubmission(ctx, submissionID)
		if err != nil {
			return err
		}
		if err := requireEnrollableSubmission(sub, agent.ID); err != nil {
			return err
		}
		now := s.now()
		entry := &model.ContestEntry{ID: s.newID(), ContestID: contestID, GameID: contest.GameID, AgentID: agent.ID,
			UserID: agent.OwnerUserID, SubmissionID: sub.ID, CreatedAt: now}
		if err := q.CreateEntry(ctx, entry); err != nil {
			if model.KindOf(err) == model.KindConflict {
				return model.Conflict("already_enrolled", "agent is already enrolled in this contest")
			}
			return err
		}
		entryID = entry.ID
		return q.Audit(ctx, p.UserID, "entry.enrolled", "contest_entry", entry.ID,
			map[string]any{"contest_id": contestID, "agent_id": agentID, "submission_id": submissionID}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetEntry(ctx, contestID, entryID)
}

// ResubmitEntry changes the locked submission while registration is open.
func (s *Service) ResubmitEntry(ctx context.Context, p model.Principal, contestID, entryID, submissionID string) (*model.ContestEntry, error) {
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		contest, err := q.ShareLockContest(ctx, contestID)
		if err != nil {
			return err
		}
		entry, err := q.LockEntry(ctx, contestID, entryID)
		if err != nil {
			return err
		}
		if !p.CanFor(entry.UserID, model.CapEntriesCreateOwn, model.CapEntriesManageAny) {
			return model.NotFound("entry_not_found", "contest entry not found")
		}
		if contest.State != model.ContestRegistrationOpen {
			return model.Conflict("roster_frozen", "the roster is frozen (state %s)", contest.State)
		}
		if entry.Status != model.EntryEnrolled {
			return model.InvalidTransition("contest entry", entry.Status, "resubmitted")
		}
		sub, err := q.LockSubmission(ctx, submissionID)
		if err != nil {
			return err
		}
		if err := requireEnrollableSubmission(sub, entry.AgentID); err != nil {
			return err
		}
		now := s.now()
		if err := q.UpdateEntrySubmission(ctx, entryID, submissionID, now); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "entry.resubmitted", "contest_entry", entryID,
			map[string]any{"before": entry.SubmissionID, "after": submissionID}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetEntry(ctx, contestID, entryID)
}

// ChangeEntryStatus withdraws (owner or manager) or disqualifies (manager)
// an entry, recording actor and reason.
func (s *Service) ChangeEntryStatus(ctx context.Context, p model.Principal, contestID, entryID string, to model.EntryStatus, reason string) (*model.ContestEntry, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 500 {
		return nil, model.Validation("invalid_reason", "a reason of 1 to 500 characters is required")
	}
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		contest, err := q.ShareLockContest(ctx, contestID)
		if err != nil {
			return err
		}
		entry, err := q.LockEntry(ctx, contestID, entryID)
		if err != nil {
			return err
		}
		switch to {
		case model.EntryWithdrawn:
			if !p.CanFor(entry.UserID, model.CapEntriesCreateOwn, model.CapEntriesManageAny) {
				return model.NotFound("entry_not_found", "contest entry not found")
			}
		case model.EntryDisqualified:
			if err := requireCap(p, model.CapEntriesManageAny); err != nil {
				return err
			}
		default:
			return model.Validation("invalid_status", "entries can only be withdrawn or disqualified")
		}
		if contest.State == model.ContestFinished || contest.State == model.ContestCancelled || contest.State == model.ContestArchived {
			return model.Conflict("contest_closed", "entries of a %s contest cannot change", contest.State)
		}
		if !entry.Status.CanTransitionTo(to) {
			return model.InvalidTransition("contest entry", entry.Status, to)
		}
		now := s.now()
		if err := q.UpdateEntryStatus(ctx, entryID, to, reason, p.UserID, now); err != nil {
			return err
		}
		if err := q.BumpRankingDirty(ctx, contestID); err != nil {
			return err
		}
		return q.Audit(ctx, p.UserID, "entry."+string(to), "contest_entry", entryID, map[string]any{"reason": reason}, now)
	})
	if err != nil {
		return nil, err
	}
	return s.store.GetEntry(ctx, contestID, entryID)
}
