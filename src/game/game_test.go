package game

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArenaBasicaEngine(t *testing.T) {
	r := require.New(t)

	registry := GetRegistry()
	engine, err := registry.CreateEngine("arena-basica")
	r.NoError(err)
	r.NotNil(engine)

	players := []string{"bot1", "bot2"}
	state, err := engine.Init(players, 12345)
	r.NoError(err)
	r.NotNil(state)
	r.Equal(0, state.Tick)
	r.False(state.Done)
	r.Equal(2, len(state.Players))
	r.Equal(100, state.Players["bot1"].HP)
	r.Equal(100, state.Players["bot2"].HP)

	// Step 1: Move bot1 Down, bot2 Up
	actions := map[string]Action{
		"bot1": {Type: ActionDown},
		"bot2": {Type: ActionUp},
	}
	nextState, err := engine.Step(actions)
	r.NoError(err)
	r.Equal(1, nextState.Tick)
	r.Equal(2, nextState.Players["bot1"].Y) // Spawned at (1,1), moving down -> (1,2)

	// Step 2: Rest & Shield
	actions = map[string]Action{
		"bot1": {Type: ActionShield},
		"bot2": {Type: ActionRest},
	}
	nextState, err = engine.Step(actions)
	r.NoError(err)
	r.True(nextState.Players["bot1"].Shielded)
	r.Equal(90, nextState.Players["bot1"].Energy) // 100 - 10

	// Step 3: Run full game until termination
	for !engine.IsOver() {
		acts := map[string]Action{
			"bot1": {Type: ActionAttack},
			"bot2": {Type: ActionAttack},
		}
		_, _ = engine.Step(acts)
	}

	r.True(engine.IsOver())
	results := engine.GetResults()
	r.NotEmpty(results)
}
