package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstants(t *testing.T) {
	r := require.New(t)

	// Context & Content Types
	r.Equal("user", UserContextKey)
	r.Equal("application/json", ContentTypeJSON)

	// Match statuses
	r.Equal("pending", MatchStatusPending)
	r.Equal("running", MatchStatusRunning)
	r.Equal("finished", MatchStatusFinished)
	r.Equal("failed", MatchStatusFailed)

	// Submission statuses
	r.Equal("pending", SubmissionStatusPending)
	r.Equal("ready", SubmissionStatusReady)
	r.Equal("failed", SubmissionStatusFailed)

	// Roles
	r.Equal("admin", RoleAdmin)
	r.Equal("participant", RoleParticipant)
	r.Equal("referee", RoleReferee)
	r.Equal("spectator", RoleSpectator)

	// Permissions
	r.Equal("invalid", InvalidPermission)
	r.Equal("read", ReadPermission)
	r.Equal("write", WritePermission)
	r.Equal("admin", AdminPermission)
	r.Equal("execute-match", ExecuteMatchPermission)
	r.Equal("submit-agent", SubmitAgentPermission)
}
