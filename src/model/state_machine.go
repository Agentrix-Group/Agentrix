package model

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidStateTransition = errors.New("invalid state transition")
)

// ContestState transitions:
// draft -> open (or published/registration_open) -> live (or in_progress/preparation) -> completed (or finished) / cancelled
func (s ContestState) CanTransitionTo(target ContestState) bool {
	switch s {
	case ContestStateDraft:
		return target == ContestStateOpen ||
			target == ContestStatePublished ||
			target == ContestStateRegistrationOpen ||
			target == ContestStateCancelled
	case ContestStatePublished:
		return target == ContestStateRegistrationOpen ||
			target == ContestStateOpen ||
			target == ContestStatePreparation ||
			target == ContestStateCancelled
	case ContestStateRegistrationOpen, ContestStateOpen:
		return target == ContestStatePreparation ||
			target == ContestStateInProgress ||
			target == ContestStateLive ||
			target == ContestStateCancelled
	case ContestStatePreparation:
		return target == ContestStateInProgress ||
			target == ContestStateLive ||
			target == ContestStateCancelled
	case ContestStateInProgress, ContestStateLive:
		return target == ContestStateFinalSelection ||
			target == ContestStateLiveFinal ||
			target == ContestStateCompleted ||
			target == ContestStateFinished ||
			target == ContestStateSuspended ||
			target == ContestStateCancelled
	case ContestStateFinalSelection:
		return target == ContestStateLiveFinal ||
			target == ContestStateCompleted ||
			target == ContestStateFinished ||
			target == ContestStateCancelled
	case ContestStateLiveFinal:
		return target == ContestStateCompleted ||
			target == ContestStateFinished ||
			target == ContestStateCancelled
	case ContestStateSuspended:
		return target == ContestStateInProgress ||
			target == ContestStateLive ||
			target == ContestStateCancelled
	case ContestStateFinished, ContestStateCompleted:
		return target == ContestStateArchived
	case ContestStateArchived, ContestStateCancelled:
		return false
	default:
		return false
	}
}

func (s ContestState) ValidateTransition(target ContestState) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("%w: cannot transition contest from %q to %q", ErrInvalidStateTransition, s, target)
	}
	return nil
}

// SubmissionStatus transitions:
// pending -> validating -> ready / failed / retired
func (s SubmissionStatus) IsValid() bool {
	switch s {
	case SubmissionStatusPending,
		SubmissionStatusValidating,
		SubmissionStatusReady,
		SubmissionStatusFailed,
		SubmissionStatusRetired:
		return true
	}
	return false
}

func (s SubmissionStatus) CanTransitionTo(target SubmissionStatus) bool {
	switch s {
	case SubmissionStatusPending:
		return target == SubmissionStatusValidating ||
			target == SubmissionStatusFailed ||
			target == SubmissionStatusRetired
	case SubmissionStatusValidating:
		return target == SubmissionStatusReady ||
			target == SubmissionStatusFailed
	case SubmissionStatusReady:
		return target == SubmissionStatusRetired
	case SubmissionStatusFailed:
		return target == SubmissionStatusRetired
	case SubmissionStatusRetired:
		return false
	default:
		return false
	}
}

func (s SubmissionStatus) ValidateTransition(target SubmissionStatus) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("%w: cannot transition submission from %q to %q", ErrInvalidStateTransition, s, target)
	}
	return nil
}

// ContestEntryStatus transitions:
// enrolled -> active -> disqualified / withdrawn
func (s ContestEntryStatus) IsValid() bool {
	switch s {
	case ContestEntryStatusEnrolled,
		ContestEntryStatusActive,
		ContestEntryStatusDisqualified,
		ContestEntryStatusWithdrawn:
		return true
	}
	return false
}

func (s ContestEntryStatus) CanTransitionTo(target ContestEntryStatus) bool {
	switch s {
	case ContestEntryStatusEnrolled:
		return target == ContestEntryStatusActive ||
			target == ContestEntryStatusDisqualified ||
			target == ContestEntryStatusWithdrawn
	case ContestEntryStatusActive:
		return target == ContestEntryStatusDisqualified ||
			target == ContestEntryStatusWithdrawn
	case ContestEntryStatusDisqualified, ContestEntryStatusWithdrawn:
		return false
	default:
		return false
	}
}

func (s ContestEntryStatus) ValidateTransition(target ContestEntryStatus) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("%w: cannot transition contest entry from %q to %q", ErrInvalidStateTransition, s, target)
	}
	return nil
}

// MatchStatus transitions:
// scheduled/pending -> queued -> running -> completed/finished / failed / cancelled
// failed -> queued (rerun allowed)
func (s MatchStatus) IsValid() bool {
	switch s {
	case MatchStatusScheduled,
		MatchStatusPending,
		MatchStatusQueued,
		MatchStatusRunning,
		MatchStatusCompleted,
		MatchStatusFinished,
		MatchStatusFailed,
		MatchStatusCancelled:
		return true
	}
	return false
}

func (s MatchStatus) CanTransitionTo(target MatchStatus) bool {
	switch s {
	case MatchStatusScheduled, MatchStatusPending:
		return target == MatchStatusQueued ||
			target == MatchStatusRunning ||
			target == MatchStatusCancelled ||
			target == MatchStatusFailed
	case MatchStatusQueued:
		return target == MatchStatusRunning ||
			target == MatchStatusCancelled ||
			target == MatchStatusFailed
	case MatchStatusFailed:
		return target == MatchStatusQueued
	case MatchStatusRunning:
		return target == MatchStatusCompleted ||
			target == MatchStatusFinished ||
			target == MatchStatusFailed ||
			target == MatchStatusCancelled
	case MatchStatusCompleted, MatchStatusFinished, MatchStatusCancelled:
		return false
	default:
		return false
	}
}

func (s MatchStatus) ValidateTransition(target MatchStatus) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("%w: cannot transition match from %q to %q", ErrInvalidStateTransition, s, target)
	}
	return nil
}

// MatchRunStatus transitions:
// created -> dispatching -> running -> completed / failed / timed_out
func (s MatchRunStatus) IsValid() bool {
	switch s {
	case MatchRunStatusCreated,
		MatchRunStatusDispatching,
		MatchRunStatusRunning,
		MatchRunStatusCompleted,
		MatchRunStatusCommitted,
		MatchRunStatusFailed,
		MatchRunStatusAborted,
		MatchRunStatusTimedOut,
		MatchRunStatusSuperseded:
		return true
	}
	return false
}

func (s MatchRunStatus) CanTransitionTo(target MatchRunStatus) bool {
	switch s {
	case MatchRunStatusCreated:
		return target == MatchRunStatusDispatching ||
			target == MatchRunStatusRunning ||
			target == MatchRunStatusFailed ||
			target == MatchRunStatusAborted ||
			target == MatchRunStatusSuperseded
	case MatchRunStatusDispatching:
		return target == MatchRunStatusRunning ||
			target == MatchRunStatusFailed ||
			target == MatchRunStatusTimedOut ||
			target == MatchRunStatusAborted
	case MatchRunStatusRunning:
		return target == MatchRunStatusCompleted ||
			target == MatchRunStatusCommitted ||
			target == MatchRunStatusFailed ||
			target == MatchRunStatusTimedOut ||
			target == MatchRunStatusAborted ||
			target == MatchRunStatusSuperseded
	case MatchRunStatusCompleted,
		MatchRunStatusCommitted,
		MatchRunStatusFailed,
		MatchRunStatusAborted,
		MatchRunStatusTimedOut,
		MatchRunStatusSuperseded:
		return false
	default:
		return false
	}
}

func (s MatchRunStatus) ValidateTransition(target MatchRunStatus) error {
	if !s.CanTransitionTo(target) {
		return fmt.Errorf("%w: cannot transition match run from %q to %q", ErrInvalidStateTransition, s, target)
	}
	return nil
}
