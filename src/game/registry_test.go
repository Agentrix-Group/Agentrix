package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryBasics(t *testing.T) {
	r := require.New(t)

	reg := NewRegistry()
	r.NotNil(reg)
	r.Empty(reg.ListManifests())

	// Test default arena-basica engine creation
	engine, err := reg.CreateEngine("arena-basica")
	r.NoError(err)
	r.NotNil(engine)
	r.Equal("arena-basica", engine.GetManifest().ID)

	// Custom manifest registration
	manifest := &Manifest{
		ID:         "custom-game",
		Name:       "Custom",
		MinPlayers: 2,
		MaxPlayers: 2,
		MaxTicks:   50,
	}
	reg.RegisterManifest(manifest)
	reg.RegisterFactory("custom-game", func(m *Manifest) Engine {
		return NewArenaBasicaEngine(m)
	})

	list := reg.ListManifests()
	r.Len(list, 1)
	r.Equal("custom-game", list[0].ID)

	customEngine, err := reg.CreateEngine("custom-game")
	r.NoError(err)
	r.NotNil(customEngine)

	// Non-existent engine
	_, err = reg.CreateEngine("non-existent")
	r.Error(err)
}

func TestArenaBasicaEngineSimulation(t *testing.T) {
	r := require.New(t)

	engine := NewArenaBasicaEngine(nil)
	r.NotNil(engine)

	// Invalid player count
	_, err := engine.Init([]string{"bot-1"}, 42)
	r.Error(err)

	// Valid init
	state, err := engine.Init([]string{"bot-1", "bot-2"}, 42)
	r.NoError(err)
	r.NotNil(state)
	r.Equal(0, state.Tick)
	r.False(engine.IsOver())

	// Step 1: Shields & Rest
	state, err = engine.Step(map[string]Action{
		"bot-1": {Type: ActionShield},
		"bot-2": {Type: ActionRest},
	})
	r.NoError(err)
	r.Equal(1, state.Tick)
	r.True(state.Players["bot-1"].Shielded)

	// Step 2: Movement
	state, err = engine.Step(map[string]Action{
		"bot-1": {Type: ActionRight},
		"bot-2": {Type: ActionLeft},
	})
	r.NoError(err)
	r.Equal(2, state.Tick)
	r.False(state.Players["bot-1"].Shielded) // shields reset next tick

	// Results map
	results := engine.GetResults()
	r.Contains(results, "bot-1")
	r.Contains(results, "bot-2")
}

func TestLoadGamesFromDir(t *testing.T) {
	r := require.New(t)

	tempDir := t.TempDir()
	gameDir := filepath.Join(tempDir, "test-game")
	err := os.MkdirAll(gameDir, 0755)
	r.NoError(err)

	manifestYAML := `
id: test-game
name: Test Game
min_players: 2
max_players: 4
max_ticks: 100
`
	err = os.WriteFile(filepath.Join(gameDir, "manifest.yaml"), []byte(manifestYAML), 0644)
	r.NoError(err)

	reg := NewRegistry()
	err = reg.LoadGamesFromDir(tempDir)
	r.NoError(err)

	manifests := reg.ListManifests()
	r.Len(manifests, 1)
	r.Equal("test-game", manifests[0].ID)
	r.Equal("Test Game", manifests[0].Name)
}
