package model

import (
	"encoding/json"
	"time"
)

type Replay struct {
	Id            string    `json:"id,omitempty" db:"id"`
	MatchId       string    `json:"match_id,omitempty" db:"match_id"`
	MatchRunId    *string   `json:"match_run_id,omitempty" db:"match_run_id"`
	FilePath      string    `json:"file_path,omitempty" db:"file_path"`
	DurationTicks int       `json:"duration_ticks,omitempty" db:"duration_ticks"`
	Summary       string    `json:"summary,omitempty" db:"summary"`
	Sha256        string    `json:"sha256,omitempty" db:"sha256"`
	SizeBytes     int64     `json:"size_bytes,omitempty" db:"size_bytes"`
	Active        bool      `json:"active,omitempty" db:"active"`
	CreatedAt     time.Time `json:"created_at,omitempty" db:"created_at"`
}

type ReplayMetadata struct {
	Type            string    `json:"type"`
	ReplayID        string    `json:"replay_id"`
	MatchID         string    `json:"match_id"`
	RunID           string    `json:"run_id,omitempty"`
	EngineDigest    string    `json:"engine_digest,omitempty"`
	GameID          string    `json:"game_id"`
	GameVersion     string    `json:"game_version,omitempty"`
	ConfigHash      string    `json:"config_hash,omitempty"`
	Seed            int64     `json:"seed"`
	Participants    []string  `json:"participants"`
	FixedTimestepMs int       `json:"fixed_timestep_ms"`
	CreatedAt       time.Time `json:"created_at"`
}

type ReplaySnapshot struct {
	Type           string          `json:"type"`
	Tick           int             `json:"tick"`
	PublicSnapshot json.RawMessage `json:"public_snapshot"`
	Events         []string        `json:"events,omitempty"`
	StateHash      string          `json:"state_hash"`
}

type ReplayResult struct {
	Type           string         `json:"type"`
	FinalTick      int            `json:"final_tick"`
	Winner         string         `json:"winner,omitempty"`
	Scores         map[string]int `json:"scores"`
	Reason         string         `json:"reason"`
	FinalStateHash string         `json:"final_state_hash"`
	FinishedAt     time.Time      `json:"finished_at"`
}

type ReplayDocument struct {
	Metadata  ReplayMetadata   `json:"metadata"`
	Snapshots []ReplaySnapshot `json:"snapshots"`
	Result    ReplayResult     `json:"result"`
}
