package game

import (
	"errors"
	"fmt"
)

func ValidateManifest(manifest *Manifest) error {
	if manifest == nil {
		return errors.New("manifest is nil")
	}
	if manifest.ID == "" {
		return errors.New("game ID is required")
	}
	if manifest.Name == "" {
		return errors.New("game name is required")
	}
	if manifest.MinPlayers <= 0 {
		return errors.New("min_players must be greater than 0")
	}
	if manifest.MaxPlayers < manifest.MinPlayers {
		return errors.New("max_players must be >= min_players")
	}
	if manifest.MaxTicks <= 0 {
		return errors.New("max_ticks must be greater than 0")
	}
	return nil
}

func ValidateAction(action *Action) error {
	if action == nil {
		return errors.New("action is nil")
	}

	switch action.Type {
	case ActionUp, ActionDown, ActionLeft, ActionRight, ActionAttack, ActionShield, ActionRest:
		return nil
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}
