package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/F4nk1/Agentrix/src/game"
)

var (
	ErrAgentUnavailable   = errors.New("agent unavailable")
	ErrAgentTimeout       = errors.New("agent timeout")
	ErrAgentExecution     = errors.New("agent execution failed")
	ErrAgentInvalidAction = errors.New("agent action invalid")
)

// OpponentPublicState exposes strictly observable arena attributes to rival bots,
// masking internal attributes such as energy, max_hp, and private strategy details (CA-004, CA-013).
type OpponentPublicState struct {
	ID       string `json:"id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Alive    bool   `json:"alive"`
	Shielded bool   `json:"shielded"`
}

// SlotPerception represents the filtered view of the arena supplied to a bot slot.
type SlotPerception struct {
	Tick       int                    `json:"tick"`
	GridWidth  int                    `json:"grid_width"`
	GridHeight int                    `json:"grid_height"`
	Me         *game.PlayerState      `json:"me,omitempty"`
	Players    map[string]interface{} `json:"players"`
	Events     []string               `json:"events"`
	Done       bool                   `json:"done"`
	Winner     string                 `json:"winner,omitempty"`
}

type Sandbox interface {
	ExecuteTurn(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error)
	ExecuteTurnWithPerception(ctx context.Context, codePath string, perception interface{}, playerID string) (game.Action, error)
	FilterPerception(state *game.GameState, playerID string) SlotPerception
}

type agentSandbox struct {
	timeout time.Duration
}

func NewSandbox(timeout time.Duration) Sandbox {
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	return &agentSandbox{timeout: timeout}
}

// FilterPerception builds a slot-isolated view of the game state for the given player.
// The bot receives its own complete state under its ID and 'me', while opponent details
// are restricted to observable coordinates and shield status.
func (s *agentSandbox) FilterPerception(state *game.GameState, playerID string) SlotPerception {
	if state == nil {
		return SlotPerception{
			Players: make(map[string]interface{}),
		}
	}

	playersMap := make(map[string]interface{})
	var meState *game.PlayerState

	for pID, p := range state.Players {
		if pID == playerID {
			meState = p
			playersMap[pID] = p
		} else {
			playersMap[pID] = OpponentPublicState{
				ID:       p.ID,
				X:        p.X,
				Y:        p.Y,
				Alive:    p.Alive,
				Shielded: p.Shielded,
			}
		}
	}

	return SlotPerception{
		Tick:       state.Tick,
		GridWidth:  state.GridWidth,
		GridHeight: state.GridHeight,
		Me:         meState,
		Players:    playersMap,
		Events:     state.Events,
		Done:       state.Done,
		Winner:     state.Winner,
	}
}

func (s *agentSandbox) ExecuteTurn(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error) {
	perception := s.FilterPerception(state, playerID)
	return s.ExecuteTurnWithPerception(ctx, codePath, perception, playerID)
}

func (s *agentSandbox) ExecuteTurnWithPerception(ctx context.Context, codePath string, perception interface{}, playerID string) (game.Action, error) {
	// If no code path is configured, attribute failure and take neutral REST action
	if codePath == "" {
		return game.Action{Type: game.ActionRest}, fmt.Errorf("%w: executable not configured", ErrAgentUnavailable)
	}

	// If script file is missing from disk, attribute failure and take neutral REST action
	if _, err := os.Stat(codePath); err != nil {
		return game.Action{Type: game.ActionRest}, fmt.Errorf("%w: %v", ErrAgentUnavailable, err)
	}

	// Execute isolated process with timeout
	action, err := s.runProcess(ctx, codePath, perception, playerID)
	if err != nil {
		return game.Action{Type: game.ActionRest}, err
	}

	// Validate action protocol compliance
	switch action.Type {
	case game.ActionUp, game.ActionDown, game.ActionLeft, game.ActionRight,
		game.ActionAttack, game.ActionShield, game.ActionRest:
		return action, nil
	default:
		return game.Action{Type: game.ActionRest}, fmt.Errorf("%w: %s", ErrAgentInvalidAction, action.Type)
	}
}

func (s *agentSandbox) runProcess(ctx context.Context, scriptPath string, perception interface{}, playerID string) (game.Action, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	perceptionJSON, err := json.Marshal(perception)
	if err != nil {
		return game.Action{Type: game.ActionRest}, fmt.Errorf("failed to marshal slot perception: %w", err)
	}

	cmd := exec.CommandContext(callCtx, "python3", scriptPath, playerID, string(perceptionJSON))
	out, err := cmd.Output()
	if err != nil {
		if callCtx.Err() != nil {
			return game.Action{Type: game.ActionRest}, fmt.Errorf("%w: %v", ErrAgentTimeout, callCtx.Err())
		}
		return game.Action{Type: game.ActionRest}, fmt.Errorf("%w: %v", ErrAgentExecution, err)
	}

	var action game.Action
	if err := json.Unmarshal(out, &action); err != nil {
		return game.Action{Type: game.ActionRest}, fmt.Errorf("%w: malformed output", ErrAgentInvalidAction)
	}

	return action, nil
}
