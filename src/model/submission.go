package model

import "time"

type Submission struct {
	Id        string    `json:"id,omitempty" db:"id"`
	AgentId   string    `json:"agent_id,omitempty" db:"agent_id"`
	Version   int       `json:"version,omitempty" db:"version"`
	CodePath  string    `json:"code_path,omitempty" db:"code_path"`
	Language  string    `json:"language,omitempty" db:"language"`
	Status    string    `json:"status,omitempty" db:"status"`
	Active    bool      `json:"active,omitempty" db:"active"`
	CreatedAt time.Time `json:"created_at,omitempty" db:"created_at"`
	Agent     *Agent    `json:"agent,omitempty"`
}
