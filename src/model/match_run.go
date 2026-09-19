package model

import "time"

type MatchRunStatus string

const (
	MatchRunStatusRunning    MatchRunStatus = "running"
	MatchRunStatusCommitted  MatchRunStatus = "committed"
	MatchRunStatusAborted    MatchRunStatus = "aborted"
	MatchRunStatusSuperseded MatchRunStatus = "superseded"
)

type MatchRun struct {
	Id           string         `json:"id" db:"id"`
	MatchId      string         `json:"match_id" db:"match_id"`
	WorkerId     string         `json:"worker_id" db:"worker_id"`
	FencingToken int64          `json:"fencing_token" db:"fencing_token"`
	Status       MatchRunStatus `json:"status" db:"status"`
	StartedAt    time.Time      `json:"started_at" db:"started_at"`
	FinishedAt   *time.Time     `json:"finished_at,omitempty" db:"finished_at"`
	HeartbeatAt  *time.Time     `json:"heartbeat_at,omitempty" db:"heartbeat_at"`
	LastError    string         `json:"last_error,omitempty" db:"last_error"`
}

type MatchResultCommit struct {
	MatchID           string    `json:"match_id"`
	RunID             string    `json:"run_id"`
	WorkerID          string    `json:"worker_id"`
	FencingToken      int64     `json:"fencing_token"`
	Status            string    `json:"status"` // e.g. finished or failed
	TerminationReason string    `json:"termination_reason,omitempty"`
	FinalTick         int       `json:"final_tick"`
	FinalStateHash    string    `json:"final_state_hash"`
	ReplayID          string    `json:"replay_id"`
	ReplaySHA256      string    `json:"replay_sha256"`
	ReplaySizeBytes   int64     `json:"replay_size_bytes"`
	ReplayPath        string    `json:"replay_path"`
	FinishedAt        time.Time `json:"finished_at"`
	Results           []Result  `json:"results"`
}
