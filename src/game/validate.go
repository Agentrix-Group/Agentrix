package game

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func ValidateManifest(manifest *Manifest) error {
	if manifest == nil {
		return errors.New("manifest is nil")
	}
	if manifest.ID != "starfighter" {
		return errors.New("Agentrix MVP supports only the starfighter game")
	}
	if manifest.Name == "" {
		return errors.New("game name is required")
	}
	if manifest.MinPlayers < model.StarfighterMinPlayers ||
		manifest.MaxPlayers > model.StarfighterMaxPlayers ||
		manifest.MinPlayers > manifest.MaxPlayers {
		return fmt.Errorf("starfighter accepts between %d and %d players",
			model.StarfighterMinPlayers, model.StarfighterMaxPlayers)
	}
	if manifest.MaxTicks <= 0 {
		return errors.New("max_ticks must be greater than 0")
	}
	if manifest.FixedTimestepMs <= 0 {
		return errors.New("fixed_timestep_ms must be greater than 0")
	}
	if manifest.BinaryPath == "" {
		return errors.New("binary_path is required")
	}
	if len(manifest.ReferenceAgents) == 0 {
		return errors.New("at least one Python reference agent is required")
	}
	for _, agent := range manifest.ReferenceAgents {
		if agent.ID == "" || !strings.HasSuffix(agent.Path, ".py") {
			return fmt.Errorf("invalid reference agent %q", agent.ID)
		}
	}
	return nil
}
