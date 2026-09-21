package model_test

import (
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestContestStateMachine(t *testing.T) {
	r := require.New(t)

	// Valid transitions
	r.NoError(model.ContestStateDraft.ValidateTransition(model.ContestStateOpen))
	r.NoError(model.ContestStateDraft.ValidateTransition(model.ContestStatePublished))
	r.NoError(model.ContestStateDraft.ValidateTransition(model.ContestStateRegistrationOpen))
	r.NoError(model.ContestStateDraft.ValidateTransition(model.ContestStateCancelled))

	r.NoError(model.ContestStateOpen.ValidateTransition(model.ContestStateLive))
	r.NoError(model.ContestStateRegistrationOpen.ValidateTransition(model.ContestStateInProgress))
	r.NoError(model.ContestStateOpen.ValidateTransition(model.ContestStateCancelled))

	r.NoError(model.ContestStateLive.ValidateTransition(model.ContestStateCompleted))
	r.NoError(model.ContestStateInProgress.ValidateTransition(model.ContestStateFinished))
	r.NoError(model.ContestStateInProgress.ValidateTransition(model.ContestStateCancelled))

	r.NoError(model.ContestStateFinished.ValidateTransition(model.ContestStateArchived))
	r.NoError(model.ContestStateCompleted.ValidateTransition(model.ContestStateArchived))

	// Invalid transitions
	r.ErrorIs(model.ContestStateDraft.ValidateTransition(model.ContestStateFinished), model.ErrInvalidStateTransition)
	r.ErrorIs(model.ContestStateFinished.ValidateTransition(model.ContestStateLive), model.ErrInvalidStateTransition)
	r.ErrorIs(model.ContestStateArchived.ValidateTransition(model.ContestStateOpen), model.ErrInvalidStateTransition)
	r.ErrorIs(model.ContestStateCancelled.ValidateTransition(model.ContestStateDraft), model.ErrInvalidStateTransition)
}

func TestSubmissionStateMachine(t *testing.T) {
	r := require.New(t)

	// Valid transitions
	r.NoError(model.SubmissionStatusPending.ValidateTransition(model.SubmissionStatusValidating))
	r.NoError(model.SubmissionStatusPending.ValidateTransition(model.SubmissionStatusFailed))
	r.NoError(model.SubmissionStatusValidating.ValidateTransition(model.SubmissionStatusReady))
	r.NoError(model.SubmissionStatusValidating.ValidateTransition(model.SubmissionStatusFailed))
	r.NoError(model.SubmissionStatusReady.ValidateTransition(model.SubmissionStatusRetired))

	// Invalid transitions
	r.ErrorIs(model.SubmissionStatusPending.ValidateTransition(model.SubmissionStatusReady), model.ErrInvalidStateTransition)
	r.ErrorIs(model.SubmissionStatusReady.ValidateTransition(model.SubmissionStatusPending), model.ErrInvalidStateTransition)
	r.ErrorIs(model.SubmissionStatusRetired.ValidateTransition(model.SubmissionStatusReady), model.ErrInvalidStateTransition)
}

func TestContestEntryStateMachine(t *testing.T) {
	r := require.New(t)

	// Valid transitions
	r.NoError(model.ContestEntryStatusEnrolled.ValidateTransition(model.ContestEntryStatusActive))
	r.NoError(model.ContestEntryStatusEnrolled.ValidateTransition(model.ContestEntryStatusDisqualified))
	r.NoError(model.ContestEntryStatusEnrolled.ValidateTransition(model.ContestEntryStatusWithdrawn))
	r.NoError(model.ContestEntryStatusActive.ValidateTransition(model.ContestEntryStatusDisqualified))
	r.NoError(model.ContestEntryStatusActive.ValidateTransition(model.ContestEntryStatusWithdrawn))

	// Invalid transitions
	r.ErrorIs(model.ContestEntryStatusDisqualified.ValidateTransition(model.ContestEntryStatusActive), model.ErrInvalidStateTransition)
	r.ErrorIs(model.ContestEntryStatusWithdrawn.ValidateTransition(model.ContestEntryStatusEnrolled), model.ErrInvalidStateTransition)
}

func TestMatchStateMachine(t *testing.T) {
	r := require.New(t)

	// Valid transitions
	r.NoError(model.MatchStatusScheduled.ValidateTransition(model.MatchStatusQueued))
	r.NoError(model.MatchStatusPending.ValidateTransition(model.MatchStatusQueued))
	r.NoError(model.MatchStatusScheduled.ValidateTransition(model.MatchStatusRunning))
	r.NoError(model.MatchStatusPending.ValidateTransition(model.MatchStatusRunning))
	r.NoError(model.MatchStatusScheduled.ValidateTransition(model.MatchStatusCancelled))
	r.NoError(model.MatchStatusScheduled.ValidateTransition(model.MatchStatusFailed))

	r.NoError(model.MatchStatusQueued.ValidateTransition(model.MatchStatusRunning))
	r.NoError(model.MatchStatusQueued.ValidateTransition(model.MatchStatusFailed))
	r.NoError(model.MatchStatusQueued.ValidateTransition(model.MatchStatusCancelled))

	// Rerun of failed match
	r.NoError(model.MatchStatusFailed.ValidateTransition(model.MatchStatusQueued))

	r.NoError(model.MatchStatusRunning.ValidateTransition(model.MatchStatusCompleted))
	r.NoError(model.MatchStatusRunning.ValidateTransition(model.MatchStatusFinished))
	r.NoError(model.MatchStatusRunning.ValidateTransition(model.MatchStatusFailed))
	r.NoError(model.MatchStatusRunning.ValidateTransition(model.MatchStatusCancelled))

	// Invalid transitions
	r.ErrorIs(model.MatchStatusScheduled.ValidateTransition(model.MatchStatusCompleted), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchStatusCompleted.ValidateTransition(model.MatchStatusRunning), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchStatusCompleted.ValidateTransition(model.MatchStatusQueued), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchStatusFinished.ValidateTransition(model.MatchStatusPending), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchStatusFinished.ValidateTransition(model.MatchStatusQueued), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchStatusFailed.ValidateTransition(model.MatchStatusRunning), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchStatusCancelled.ValidateTransition(model.MatchStatusRunning), model.ErrInvalidStateTransition)
}

func TestMatchRunStateMachine(t *testing.T) {
	r := require.New(t)

	// Valid transitions
	r.NoError(model.MatchRunStatusCreated.ValidateTransition(model.MatchRunStatusDispatching))
	r.NoError(model.MatchRunStatusCreated.ValidateTransition(model.MatchRunStatusRunning))
	r.NoError(model.MatchRunStatusDispatching.ValidateTransition(model.MatchRunStatusRunning))
	r.NoError(model.MatchRunStatusDispatching.ValidateTransition(model.MatchRunStatusTimedOut))
	r.NoError(model.MatchRunStatusDispatching.ValidateTransition(model.MatchRunStatusFailed))

	r.NoError(model.MatchRunStatusRunning.ValidateTransition(model.MatchRunStatusCompleted))
	r.NoError(model.MatchRunStatusRunning.ValidateTransition(model.MatchRunStatusCommitted))
	r.NoError(model.MatchRunStatusRunning.ValidateTransition(model.MatchRunStatusFailed))
	r.NoError(model.MatchRunStatusRunning.ValidateTransition(model.MatchRunStatusTimedOut))
	r.NoError(model.MatchRunStatusRunning.ValidateTransition(model.MatchRunStatusSuperseded))

	// Invalid transitions
	r.ErrorIs(model.MatchRunStatusCreated.ValidateTransition(model.MatchRunStatusCompleted), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchRunStatusCompleted.ValidateTransition(model.MatchRunStatusRunning), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchRunStatusFailed.ValidateTransition(model.MatchRunStatusRunning), model.ErrInvalidStateTransition)
	r.ErrorIs(model.MatchRunStatusTimedOut.ValidateTransition(model.MatchRunStatusRunning), model.ErrInvalidStateTransition)
}
