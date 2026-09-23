package common

const (
	UserContextKey  = "user"
	ContentTypeJSON = "application/json"

	// Match statuses
	MatchStatusScheduled = "scheduled"
	MatchStatusPending   = "pending"
	MatchStatusQueued    = "queued"
	MatchStatusRunning   = "running"
	MatchStatusFinished  = "finished"
	MatchStatusFailed    = "failed"

	// Submission statuses
	SubmissionStatusPending = "pending"
	SubmissionStatusReady   = "ready"
	SubmissionStatusFailed  = "failed"
	// Admisión asíncrona (ADR-0014, N4): el bot espera la prueba del worker.
	SubmissionStatusValidating = "validating"
	SubmissionStatusRejected   = "rejected"

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
