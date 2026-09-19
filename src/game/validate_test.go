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
	invalid = valid
	invalid.MaxPlayers = 4
	r.Error(ValidateManifest(&invalid))
	invalid = valid
	invalid.FixedTimestepMs = 0
	r.Error(ValidateManifest(&invalid))
	invalid = valid
	invalid.ReferenceAgents = nil
	r.Error(ValidateManifest(&invalid))
}
