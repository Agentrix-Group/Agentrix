package game

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateManifest(t *testing.T) {
	r := require.New(t)

	// nil manifest
	r.Error(ValidateManifest(nil))

	valid := Manifest{
		ID: "starfighter", Name: "Starfighter Arena", MinPlayers: 2, MaxPlayers: 2,
		MaxTicks: 100, FixedTimestepMs: 17, BinaryPath: "bin/starfighter-engine",
		ReferenceAgents: []ReferenceAgent{{ID: "hunter", Path: "bot_hunter.py"}},
	}
	r.NoError(ValidateManifest(&valid))

	invalid := valid
	invalid.ID = "other-game"
	r.Error(ValidateManifest(&invalid))
	// ADR-0013: Starfighter admite de 2 a 5 jugadores.
	for _, maxPlayers := range []int{3, 4, 5} {
		ffa := valid
		ffa.MaxPlayers = maxPlayers
		r.NoError(ValidateManifest(&ffa), "max_players=%d", maxPlayers)
	}
	invalid = valid
	invalid.MaxPlayers = 6
	r.Error(ValidateManifest(&invalid))
	invalid = valid
	invalid.MinPlayers = 1
	r.Error(ValidateManifest(&invalid))
	invalid = valid
	invalid.MinPlayers = 4
	invalid.MaxPlayers = 3
	r.Error(ValidateManifest(&invalid))
	invalid = valid
	invalid.FixedTimestepMs = 0
	r.Error(ValidateManifest(&invalid))
	invalid = valid
	invalid.ReferenceAgents = nil
	r.Error(ValidateManifest(&invalid))
}
