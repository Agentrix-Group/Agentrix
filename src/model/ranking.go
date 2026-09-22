package model

import "time"

type Ranking struct {
	Id                string    `json:"id,omitempty" db:"id"`
	ContestId         string    `json:"contest_id,omitempty" db:"contest_id"`
	AgentId           string    `json:"agent_id,omitempty" db:"agent_id"`
	UserId            string    `json:"user_id,omitempty" db:"user_id"`
	Score             int       `json:"score,omitempty" db:"score"`
	Points            int       `json:"points,omitempty" db:"points"`
	MatchesPlayed     int       `json:"matches_played,omitempty" db:"matches_played"`
	Wins              int       `json:"wins,omitempty" db:"wins"`
	Losses            int       `json:"losses,omitempty" db:"losses"`
	Draws             int       `json:"draws,omitempty" db:"draws"`
	Disqualifications int       `json:"disqualifications,omitempty" db:"disqualifications"`
	TiebreakerScore   float64   `json:"tiebreaker_score,omitempty" db:"tiebreaker_score"`
	Rank              int       `json:"rank,omitempty" db:"rank"`
	UpdatedAt         time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Agent             *Agent    `json:"agent,omitempty"`
	User              *User     `json:"user,omitempty"`
}

type RankingSnapshot struct {
	Id          string    `json:"id" db:"id"`
	ContestId   string    `json:"contest_id" db:"contest_id"`
	Version     int       `json:"version" db:"version"`
	Rankings    []Ranking `json:"rankings"`
	PublishedBy *string   `json:"published_by,omitempty" db:"published_by"`
	PublishedAt time.Time `json:"published_at" db:"published_at"`
}
