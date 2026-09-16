package game

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateManifest(t *testing.T) {
	r := require.New(t)

	// nil manifest
	r.Error(ValidateManifest(nil))

	// missing ID
	r.Error(ValidateManifest(&Manifest{Name: "Game", MinPlayers: 2, MaxPlayers: 4, MaxTicks: 100}))

	// missing Name
	r.Error(ValidateManifest(&Manifest{ID: "g1", MinPlayers: 2, MaxPlayers: 4, MaxTicks: 100}))

	// invalid min players
	r.Error(ValidateManifest(&Manifest{ID: "g1", Name: "Game", MinPlayers: 0, MaxPlayers: 4, MaxTicks: 100}))

	// max players < min players
	r.Error(ValidateManifest(&Manifest{ID: "g1", Name: "Game", MinPlayers: 4, MaxPlayers: 2, MaxTicks: 100}))

	// invalid max ticks
	r.Error(ValidateManifest(&Manifest{ID: "g1", Name: "Game", MinPlayers: 2, MaxPlayers: 4, MaxTicks: 0}))

	// valid manifest
	r.NoError(ValidateManifest(&Manifest{
		ID:         "arena-basica",
		Name:       "Arena Basica",
		MinPlayers: 2,
		MaxPlayers: 4,
		MaxTicks:   100,
	}))
}

func TestValidateAction(t *testing.T) {
	r := require.New(t)

	// nil action
	r.Error(ValidateAction(nil))

	// valid actions
	validActions := []ActionType{
		ActionUp,
		ActionDown,
		ActionLeft,
		ActionRight,
		ActionAttack,
		ActionShield,
		ActionRest,
	}

	for _, actType := range validActions {
		r.NoError(ValidateAction(&Action{Type: actType}))
	}

	// invalid action
	r.Error(ValidateAction(&Action{Type: "teleport"}))
	r.Error(ValidateAction(&Action{Type: ""}))
}
