package model

import "time"

type Match struct {
	Id          string       `json:"id,omitempty" db:"id"`
	ContestId   string       `json:"contest_id,omitempty" db:"contest_id"`
	GameId      string       `json:"game_id,omitempty" db:"game_id"`
	Status      string       `json:"status,omitempty" db:"status"`
	Seed        int64        `json:"seed,omitempty" db:"seed"`
	ReplayId    string       `json:"replay_id,omitempty" db:"replay_id"`
	Active      bool         `json:"active,omitempty" db:"active"`
	CreatedAt   time.Time    `json:"created_at,omitempty" db:"created_at"`
	FinishedAt  *time.Time   `json:"finished_at,omitempty" db:"finished_at"`
	Results     []Result     `json:"results,omitempty"`
	Submissions []Submission `json:"submissions,omitempty"`
}
