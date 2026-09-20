package model

import "time"

type Ranking struct {
	Id            string    `json:"id,omitempty" db:"id"`
	ContestId     string    `json:"contest_id,omitempty" db:"contest_id"`
	AgentId       string    `json:"agent_id,omitempty" db:"agent_id"`
	UserId        string    `json:"user_id,omitempty" db:"user_id"`
	ParticipantId string    `json:"participant_id,omitempty"` // Backward compatibility alias
	Score         int       `json:"score,omitempty" db:"score"`
	MatchesPlayed int       `json:"matches_played,omitempty" db:"matches_played"`
	Wins          int       `json:"wins,omitempty" db:"wins"`
	Losses        int       `json:"losses,omitempty" db:"losses"`
	Draws         int       `json:"draws,omitempty" db:"draws"`
	Rank          int       `json:"rank,omitempty" db:"rank"`
	UpdatedAt     time.Time `json:"updated_at,omitempty" db:"updated_at"`
	Agent         *Agent    `json:"agent,omitempty"`
	User          *User     `json:"user,omitempty"`
	Participant   *User     `json:"participant,omitempty"` // Backward compatibility alias
}
