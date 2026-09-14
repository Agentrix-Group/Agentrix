package common

const (
	UserContextKey  = "user"
	ContentTypeJSON = "application/json"

	// Match statuses
	MatchStatusPending  = "pending"
	MatchStatusRunning  = "running"
	MatchStatusFinished = "finished"
	MatchStatusFailed   = "failed"

	// Submission statuses
	SubmissionStatusPending = "pending"
	SubmissionStatusReady   = "ready"
	SubmissionStatusFailed  = "failed"

	// Roles
	RoleAdmin       = "admin"
	RoleParticipant = "participant"
	RoleReferee     = "referee"
	RoleSpectator   = "spectator"

	// Permissions
	InvalidPermission      = "invalid"
	ReadPermission         = "read"
	WritePermission        = "write"
	AdminPermission        = "admin"
	ExecuteMatchPermission = "execute-match"
	SubmitAgentPermission  = "submit-agent"
)
