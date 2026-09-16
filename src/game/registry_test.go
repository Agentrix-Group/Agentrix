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

	// Custom manifest registration
	manifest := &Manifest{
		ID:         "custom-game",
		Name:       "Custom",
		MinPlayers: 2,
		MaxPlayers: 2,
		MaxTicks:   50,
	}
	reg.RegisterManifest(manifest)

	list := reg.ListManifests()
	r.Len(list, 1)
	r.Equal("custom-game", list[0].ID)

	retrieved := reg.GetManifest("custom-game")
	r.NotNil(retrieved)
	r.Equal("Custom", retrieved.Name)

	nonExistent := reg.GetManifest("non-existent")
	r.Nil(nonExistent)
}

func TestGetRegistrySingleton(t *testing.T) {
	r := require.New(t)

	reg1 := GetRegistry()
	reg2 := GetRegistry()
	r.NotNil(reg1)
	r.Same(reg1, reg2)
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

func TestLoadGamesFromDir_InvalidDir(t *testing.T) {
	r := require.New(t)

	reg := NewRegistry()
	err := reg.LoadGamesFromDir("/path/to/nonexistent/directory")
	r.Error(err)
}
