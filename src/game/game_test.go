package game

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestManifestSerialization(t *testing.T) {
	r := require.New(t)

	manifestYAML := `
id: arena-basica
name: Arena Basica
version: 1.0.0
description: A grid battle game
min_players: 2
max_players: 4
max_ticks: 100
grid_width: 10
grid_height: 10
binary_path: bin/fake-engine
settings:
  initial_hp: "100"
`
	var m Manifest
	err := yaml.Unmarshal([]byte(manifestYAML), &m)
	r.NoError(err)
	r.Equal("arena-basica", m.ID)
	r.Equal("Arena Basica", m.Name)
	r.Equal(2, m.MinPlayers)
	r.Equal(4, m.MaxPlayers)
	r.Equal(100, m.MaxTicks)
	r.Equal("bin/fake-engine", m.BinaryPath)
	r.Equal("100", m.Settings["initial_hp"])

	jsonBytes, err := json.Marshal(m)
	r.NoError(err)
	r.Contains(string(jsonBytes), `"id":"arena-basica"`)
}

func TestActionTypes(t *testing.T) {
	r := require.New(t)

	r.Equal(ActionType("UP"), ActionUp)
	r.Equal(ActionType("DOWN"), ActionDown)
	r.Equal(ActionType("LEFT"), ActionLeft)
	r.Equal(ActionType("RIGHT"), ActionRight)
	r.Equal(ActionType("ATTACK"), ActionAttack)
	r.Equal(ActionType("SHIELD"), ActionShield)
	r.Equal(ActionType("REST"), ActionRest)

	act := Action{
		Type: ActionAttack,
		Payload: map[string]interface{}{
			"target": "bot-2",
		},
	}
	bytes, err := json.Marshal(act)
	r.NoError(err)
	r.Contains(string(bytes), `"type":"ATTACK"`)
}

func TestGameStateSerialization(t *testing.T) {
	r := require.New(t)

	state := GameState{
		Tick:       1,
		GridWidth:  10,
		GridHeight: 10,
		Players: map[string]*PlayerState{
			"bot-1": {
				ID:       "bot-1",
				X:        1,
				Y:        1,
				HP:       100,
				MaxHP:    100,
				Energy:   100,
				Shielded: false,
				Alive:    true,
				Score:    0,
			},
		},
		Events: []string{"Game started"},
		Done:   false,
	}

	bytes, err := json.Marshal(state)
	r.NoError(err)
	r.Contains(string(bytes), `"tick":1`)
	r.Contains(string(bytes), `"bot-1"`)
}
