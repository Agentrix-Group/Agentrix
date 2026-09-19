package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGameJSONSerialization(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	game := Game{
		Id:           "starfighter",
		Name:         "Starfighter Arena",
		Description:  "2 to 4 player grid battle",
		ManifestPath: "/games/starfighter/manifest.yaml",
		MinPlayers:   2,
		MaxPlayers:   4,
		Active:       true,
		CreatedAt:    now,
	}

	data, err := json.Marshal(game)
	r.NoError(err)
	r.Contains(string(data), `"id":"starfighter"`)
	r.Contains(string(data), `"min_players":2`)
	r.Contains(string(data), `"max_players":4`)

	var parsed Game
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("starfighter", parsed.Id)
	r.Equal(2, parsed.MinPlayers)
	r.Equal(4, parsed.MaxPlayers)
	r.True(parsed.Active)
}
