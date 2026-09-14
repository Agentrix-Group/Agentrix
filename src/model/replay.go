package model

import "time"

type ReplayFrame struct {
	Tick    int                    `json:"tick"`
	Events  []string               `json:"events,omitempty"`
	State   map[string]interface{} `json:"state"`
	Actions map[string]interface{} `json:"actions,omitempty"`
}

type ReplayData struct {
	GameId   string                 `json:"game_id"`
	MatchId  string                 `json:"match_id"`
	Seed     int64                  `json:"seed"`
	Players  []string               `json:"players"`
	MaxTicks int                    `json:"max_ticks"`
	Frames   []ReplayFrame          `json:"frames"`
	Winner   string                 `json:"winner,omitempty"`
	Scores   map[string]int         `json:"scores"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

type Replay struct {
	Id            string      `json:"id,omitempty" db:"id"`
	MatchId       string      `json:"match_id,omitempty" db:"match_id"`
	FilePath      string      `json:"file_path,omitempty" db:"file_path"`
	DurationTicks int         `json:"duration_ticks,omitempty" db:"duration_ticks"`
	Summary       string      `json:"summary,omitempty" db:"summary"`
	Active        bool        `json:"active,omitempty" db:"active"`
	CreatedAt     time.Time   `json:"created_at,omitempty" db:"created_at"`
	Data          *ReplayData `json:"data,omitempty"`
}
