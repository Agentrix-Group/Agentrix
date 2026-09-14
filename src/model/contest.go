package model

import "time"

type Contest struct {
	Id          string    `json:"id,omitempty" db:"id"`
	Name        string    `json:"name,omitempty" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	GameId      string    `json:"game_id,omitempty" db:"game_id"`
	CategoryId  string    `json:"category_id,omitempty" db:"category_id"`
	StartDate   time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate     time.Time `json:"end_date,omitempty" db:"end_date"`
	Status      string    `json:"status,omitempty" db:"status"`
	Active      bool      `json:"active,omitempty" db:"active"`
	CreatedAt   time.Time `json:"created_at,omitempty" db:"created_at"`
}
