package model

import "time"

type Game struct {
	Id           string    `json:"id,omitempty" db:"id"`
	Name         string    `json:"name,omitempty" db:"name"`
	Description  string    `json:"description,omitempty" db:"description"`
	ManifestPath string    `json:"manifest_path,omitempty" db:"manifest_path"`
	MinPlayers   int       `json:"min_players,omitempty" db:"min_players"`
	MaxPlayers   int       `json:"max_players,omitempty" db:"max_players"`
	Active       bool      `json:"active,omitempty" db:"active"`
	CreatedAt    time.Time `json:"created_at,omitempty" db:"created_at"`
}
