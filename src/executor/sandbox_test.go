package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/game"
	"github.com/stretchr/testify/require"
)

func TestFilterPerception(t *testing.T) {
	r := require.New(t)
	sandbox := NewSandbox(100 * time.Millisecond)

	gameState := &game.GameState{
		Tick:       5,
		GridWidth:  10,
		GridHeight: 10,
		Players: map[string]*game.PlayerState{
			"bot-1": {
				ID:       "bot-1",
				X:        2,
				Y:        3,
				HP:       80,
				MaxHP:    100,
				Energy:   65,
				Shielded: false,
				Alive:    true,
				Score:    25,
			},
			"bot-2": {
				ID:       "bot-2",
				X:        4,
				Y:        5,
				HP:       40,
				MaxHP:    100,
				Energy:   90,
				Shielded: true,
				Alive:    true,
				Score:    10,
			},
		},
		Events: []string{"bot-1 moved to (2,3)"},
	}

	// Slot perception for bot-1
	p1 := sandbox.FilterPerception(gameState, "bot-1")
	r.Equal(5, p1.Tick)
	r.Equal(10, p1.GridWidth)
	r.NotNil(p1.Me)
	r.Equal("bot-1", p1.Me.ID)
	r.Equal(65, p1.Me.Energy)
	r.Equal(80, p1.Me.HP)

	// In players map: bot-1 has full PlayerState
	mePlayer, ok := p1.Players["bot-1"].(*game.PlayerState)
	r.True(ok)
	r.Equal(65, mePlayer.Energy)

	// Opponent bot-2 should only expose public observable fields
	oppPlayer, ok := p1.Players["bot-2"].(OpponentPublicState)
	r.True(ok)
	r.Equal("bot-2", oppPlayer.ID)
	r.Equal(4, oppPlayer.X)
	r.Equal(5, oppPlayer.Y)
	r.True(oppPlayer.Alive)
	r.True(oppPlayer.Shielded)
}

func TestExecuteTurn_AttributableFailures(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	sandbox := NewSandbox(500 * time.Millisecond)

	state := &game.GameState{
		Tick:       1,
		GridWidth:  10,
		GridHeight: 10,
		Players: map[string]*game.PlayerState{
			"bot-1": {
				ID:    "bot-1",
				X:     1,
				Y:     1,
				HP:    100,
				Alive: true,
			},
		},
	}

	t.Run("Empty code path returns ActionRest and attributable error", func(t *testing.T) {
		action, err := sandbox.ExecuteTurn(ctx, "", state, "bot-1")
		r.Error(err)
		r.Equal(game.ActionRest, action.Type)
	})

	t.Run("Missing script file returns ActionRest and attributable error", func(t *testing.T) {
		action, err := sandbox.ExecuteTurn(ctx, "/path/to/nonexistent/script.py", state, "bot-1")
		r.Error(err)
		r.Equal(game.ActionRest, action.Type)
	})

	t.Run("Crashing script returns ActionRest and attributable error without heuristic fallback", func(t *testing.T) {
		tmpDir := t.TempDir()
		badScript := filepath.Join(tmpDir, "crash.py")
		err := os.WriteFile(badScript, []byte("import sys\nsys.exit(1)\n"), 0755)
		r.NoError(err)

		action, err := sandbox.ExecuteTurn(ctx, badScript, state, "bot-1")
		r.Error(err)
		r.Equal(game.ActionRest, action.Type)
	})

	t.Run("Invalid JSON script returns ActionRest and attributable error", func(t *testing.T) {
		tmpDir := t.TempDir()
		badScript := filepath.Join(tmpDir, "invalid_json.py")
		err := os.WriteFile(badScript, []byte("print('this is not json')\n"), 0755)
		r.NoError(err)

		action, err := sandbox.ExecuteTurn(ctx, badScript, state, "bot-1")
		r.Error(err)
		r.Equal(game.ActionRest, action.Type)
	})

	t.Run("Unrecognized action type defaults to ActionRest", func(t *testing.T) {
		tmpDir := t.TempDir()
		badScript := filepath.Join(tmpDir, "invalid_action.py")
		err := os.WriteFile(badScript, []byte("import json\nprint(json.dumps({'type': 'TELEPORT'}))\n"), 0755)
		r.NoError(err)

		action, err := sandbox.ExecuteTurn(ctx, badScript, state, "bot-1")
		r.Error(err)
		r.Equal(game.ActionRest, action.Type)
	})
}

func TestExecuteTurn_ValidBotExecution(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	sandbox := NewSandbox(2 * time.Second)

	scriptPath := "../../games/arena-basica/examples/bot_hunter.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		scriptPath = "games/arena-basica/examples/bot_hunter.py"
	}

	state := &game.GameState{
		Tick:       1,
		GridWidth:  10,
		GridHeight: 10,
		Players: map[string]*game.PlayerState{
			"bot-1": {
				ID:       "bot-1",
				X:        1,
				Y:        1,
				HP:       100,
				MaxHP:    100,
				Energy:   100,
				Shielded: false,
				Alive:    true,
				Score:    0,
			},
			"bot-2": {
				ID:       "bot-2",
				X:        8,
				Y:        8,
				HP:       100,
				MaxHP:    100,
				Energy:   100,
				Shielded: false,
				Alive:    true,
				Score:    0,
			},
		},
	}

	action, err := sandbox.ExecuteTurn(ctx, scriptPath, state, "bot-1")
	r.NoError(err)
	r.Contains([]game.ActionType{
		game.ActionUp, game.ActionDown, game.ActionLeft, game.ActionRight,
		game.ActionAttack, game.ActionShield, game.ActionRest,
	}, action.Type)
}
