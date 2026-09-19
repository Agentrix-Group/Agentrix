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
id: starfighter
name: Starfighter Arena
version: 1.0.0
description: A deterministic space duel
min_players: 2
max_players: 2
max_ticks: 100
binary_path: bin/fake-engine
fixed_timestep_ms: 17
reference_agents:
  - id: hunter
    path: games/starfighter/examples/bot_hunter.py
settings:
  initial_hp: "100"
`
	var m Manifest
	err := yaml.Unmarshal([]byte(manifestYAML), &m)
	r.NoError(err)
	r.Equal("starfighter", m.ID)
	r.Equal("Starfighter Arena", m.Name)
	r.Equal(2, m.MinPlayers)
	r.Equal(2, m.MaxPlayers)
	r.Equal(100, m.MaxTicks)
	r.Equal(17, m.FixedTimestepMs)
	r.Equal("hunter", m.ReferenceAgents[0].ID)
	r.Equal("bin/fake-engine", m.BinaryPath)
	r.Equal("100", m.Settings["initial_hp"])

	jsonBytes, err := json.Marshal(m)
	r.NoError(err)
	r.Contains(string(jsonBytes), `"id":"starfighter"`)
}
