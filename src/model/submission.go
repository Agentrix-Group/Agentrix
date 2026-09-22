package model

import "time"

type SubmissionStatus string

const (
	SubmissionStatusPending    SubmissionStatus = "pending"
	SubmissionStatusValidating SubmissionStatus = "validating"
	SubmissionStatusReady      SubmissionStatus = "ready"
	SubmissionStatusFailed     SubmissionStatus = "failed"
	SubmissionStatusRetired    SubmissionStatus = "retired"
)

type Submission struct {
	Id          string    `json:"id,omitempty" db:"id"`
	AgentId     string    `json:"agent_id,omitempty" db:"agent_id"`
	Version     int       `json:"version,omitempty" db:"version"`
	CodePath    string    `json:"-" db:"code_path"`
	Language    string    `json:"language,omitempty" db:"language"`
	Status      string    `json:"status,omitempty" db:"status"`
	Active      bool      `json:"active,omitempty" db:"active"`
	ErrorDetail string    `json:"error_detail,omitempty" db:"-"`
	CreatedAt   time.Time `json:"created_at,omitempty" db:"created_at"`
	Agent       *Agent    `json:"agent,omitempty"`
}

type AgentPackageManifest struct {
	Name            string `json:"name"`
	Entrypoint      string `json:"entrypoint"`
	ProtocolVersion string `json:"protocol_version"`
}
