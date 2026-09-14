package executor

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/tracer"
)

type Sandbox interface {
	ExecuteTurn(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error)
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

func (s *agentSandbox) ExecuteTurn(ctx context.Context, codePath string, state *game.GameState, playerID string) (game.Action, error) {
	// If script file exists on disk, execute with timeout and JSON IPC
	if codePath != "" {
		if _, err := os.Stat(codePath); err == nil {
			action, err := s.runProcess(ctx, codePath, state, playerID)
			if err == nil {
				return action, nil
			}
			tracer.Warnf(ctx, "Bot process execution failed, falling back to heuristic: %s", err)
		}
	}

	// Default intelligent heuristic bot behavior for Arena Basica
	return s.heuristicBot(state, playerID), nil
}

func (s *agentSandbox) runProcess(ctx context.Context, scriptPath string, state *game.GameState, playerID string) (game.Action, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return game.Action{Type: game.ActionRest}, err
	}

	cmd := exec.CommandContext(callCtx, "python3", scriptPath, playerID, string(stateJSON))
	out, err := cmd.Output()
	if err != nil {
		return game.Action{Type: game.ActionRest}, err
	}

	var action game.Action
	if err := json.Unmarshal(out, &action); err != nil {
		return game.Action{Type: game.ActionRest}, err
	}

	return action, nil
}

func (s *agentSandbox) heuristicBot(state *game.GameState, playerID string) game.Action {
	me, ok := state.Players[playerID]
	if !ok || !me.Alive {
		return game.Action{Type: game.ActionRest}
	}

	// Find closest opponent
	var closestOpponent *game.PlayerState
	minDist := 9999

	for pID, opp := range state.Players {
		if pID == playerID || !opp.Alive {
			continue
		}
		dist := absDiff(me.X, opp.X) + absDiff(me.Y, opp.Y)
		if dist < minDist {
			minDist = dist
			closestOpponent = opp
		}
	}

	if closestOpponent == nil {
		return game.Action{Type: game.ActionRest}
	}

	// If adjacent, attack or shield
	if minDist <= 1 {
		if me.HP < 25 && me.Energy >= 10 {
			return game.Action{Type: game.ActionShield}
		}
		if me.Energy >= 15 {
			return game.Action{Type: game.ActionAttack}
		}
		return game.Action{Type: game.ActionRest}
	}

	// If low on energy, rest
	if me.Energy < 20 {
		return game.Action{Type: game.ActionRest}
	}

	// Navigate towards closest opponent
	dx := closestOpponent.X - me.X
	dy := closestOpponent.Y - me.Y

	if absDiff(dx, 0) > absDiff(dy, 0) {
		if dx > 0 {
			return game.Action{Type: game.ActionRight}
		}
		return game.Action{Type: game.ActionLeft}
	} else {
		if dy > 0 {
			return game.Action{Type: game.ActionDown}
		}
		return game.Action{Type: game.ActionUp}
	}
}

func absDiff(a, b int) int {
	diff := a - b
	if diff < 0 {
		return -diff
	}
	return diff
}

func randomMove() game.Action {
	moves := []game.ActionType{
		game.ActionUp, game.ActionDown, game.ActionLeft, game.ActionRight,
		game.ActionAttack, game.ActionShield, game.ActionRest,
	}
	idx := rand.Intn(len(moves))
	return game.Action{Type: moves[idx]}
}
