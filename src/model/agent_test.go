package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAgentJSONSerialization(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	agent := Agent{
		Id:          "agent-1",
		OwnerUserId: "user-1",
		GameId:      "game-1",
		Name:        "AlphaBot",
		Description: "First test agent",
		Active:      true,
		CreatedAt:   now,
		OwnerUser: &User{
			Id:       "user-1",
			Username: "player1",
		},
		Game: &Game{
			Id:   "game-1",
			Name: "Arena",
		},
	}

	data, err := json.Marshal(agent)
	r.NoError(err)
	r.Contains(string(data), `"id":"agent-1"`)
	r.Contains(string(data), `"name":"AlphaBot"`)
	r.Contains(string(data), `"active":true`)

	var parsed Agent
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("agent-1", parsed.Id)
	r.Equal("AlphaBot", parsed.Name)
	r.True(parsed.Active)
	r.NotNil(parsed.OwnerUser)
	r.Equal("player1", parsed.OwnerUser.Username)
	r.NotNil(parsed.Game)
	r.Equal("Arena", parsed.Game.Name)
}
