package model

import "time"

type MatchSlot struct {
	Id             string    `json:"id" db:"id"`
	MatchId        string    `json:"match_id" db:"match_id"`
	SlotIndex      int       `json:"slot_index" db:"slot_index"`
	ContestEntryId *string   `json:"contest_entry_id,omitempty" db:"contest_entry_id"`
	SubmissionId   string    `json:"submission_id" db:"submission_id"`
	AgentName      string    `json:"agent_name,omitempty" db:"agent_name"`
	Username       string    `json:"username,omitempty" db:"username"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type Match struct {
	Id             string       `json:"id,omitempty" db:"id"`
	ContestId      string       `json:"contest_id,omitempty" db:"contest_id"`
	GameId         string       `json:"game_id,omitempty" db:"game_id"`
	RunId          string       `json:"run_id,omitempty" db:"run_id"`
	CommittedRunId *string      `json:"committed_run_id,omitempty" db:"committed_run_id"`
	GameVersion    string       `json:"game_version,omitempty" db:"game_version"`
	EngineDigest   string       `json:"engine_digest,omitempty" db:"engine_digest"`
	ConfigHash     string       `json:"config_hash,omitempty" db:"config_hash"`
	Status         string       `json:"status,omitempty" db:"status"`
	Seed           int64        `json:"seed,omitempty" db:"seed"`
	ReplayId       string       `json:"replay_id,omitempty" db:"replay_id"`
	Active         bool         `json:"active,omitempty" db:"active"`
	CreatedAt      time.Time    `json:"created_at,omitempty" db:"created_at"`
	FinishedAt     *time.Time   `json:"finished_at,omitempty" db:"finished_at"`
	Slots          []MatchSlot  `json:"slots,omitempty"`
	Results        []Result     `json:"results,omitempty"`
	Submissions    []Submission `json:"submissions,omitempty"`
}

type MatchStatus string

const (
	MatchStatusScheduled MatchStatus = "scheduled"
	MatchStatusPending   MatchStatus = "pending"
	MatchStatusQueued    MatchStatus = "queued"
	MatchStatusRunning   MatchStatus = "running"
	MatchStatusCompleted MatchStatus = "completed"
	MatchStatusFinished  MatchStatus = "finished"
	MatchStatusFailed    MatchStatus = "failed"
	MatchStatusCancelled MatchStatus = "cancelled"
)

type MatchResponse struct {
	HttpStatusCode int         `json:"http_status_code,omitempty"`
	Message        string      `json:"message,omitempty"`
	MatchId        string      `json:"match_id"`
	Match          *Match      `json:"match,omitempty"`
	Slots          []MatchSlot `json:"slots,omitempty"`
}

type RunMatchResponse struct {
	HttpStatusCode int    `json:"-"`
	MatchId        string `json:"match_id"`
	RunId          string `json:"run_id"`
	JobId          string `json:"job_id"`
	Status         string `json:"status"`
}
