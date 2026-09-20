package model

import "time"

type Agent struct {
	Id            string    `json:"id,omitempty" db:"id"`
	OwnerUserId   string    `json:"owner_user_id,omitempty" db:"owner_user_id"`
	ParticipantId string    `json:"participant_id,omitempty"` // Backward compatibility alias
	GameId        string    `json:"game_id,omitempty" db:"game_id"`
	Name          string    `json:"name,omitempty" db:"name"`
	Description   string    `json:"description,omitempty" db:"description"`
	Active        bool      `json:"active,omitempty" db:"active"`
	CreatedAt     time.Time `json:"created_at,omitempty" db:"created_at"`
	OwnerUser     *User     `json:"owner_user,omitempty"`
	Participant   *User     `json:"participant,omitempty"` // Backward compatibility alias
	Game          *Game     `json:"game,omitempty"`
}
