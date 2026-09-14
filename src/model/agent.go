package model

import "time"

type Agent struct {
	Id            string       `json:"id,omitempty" db:"id"`
	ParticipantId string       `json:"participant_id,omitempty" db:"participant_id"`
	GameId        string       `json:"game_id,omitempty" db:"game_id"`
	Name          string       `json:"name,omitempty" db:"name"`
	Description   string       `json:"description,omitempty" db:"description"`
	Active        bool         `json:"active,omitempty" db:"active"`
	CreatedAt     time.Time    `json:"created_at,omitempty" db:"created_at"`
	Participant   *Participant `json:"participant,omitempty"`
	Game          *Game        `json:"game,omitempty"`
}
