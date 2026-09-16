package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReplayDataAndFrameJSON(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	frame := ReplayFrame{
		Tick:   1,
		Events: []string{"spawn:bot-1", "spawn:bot-2"},
		State: map[string]interface{}{
			"turn": 1,
		},
		Actions: map[string]interface{}{
			"bot-1": "move:up",
		},
	}

	replayData := ReplayData{
		GameId:   "arena-basica",
		MatchId:  "match-100",
		Seed:     42,
		Players:  []string{"bot-1", "bot-2"},
		MaxTicks: 500,
		Frames:   []ReplayFrame{frame},
		Winner:   "bot-1",
		Scores:   map[string]int{"bot-1": 100, "bot-2": 50},
	}

	replay := Replay{
		Id:            "replay-1",
		MatchId:       "match-100",
		FilePath:      "/storage/replays/match-100.json",
		DurationTicks: 250,
		Summary:       "bot-1 eliminated bot-2",
		Active:        true,
		CreatedAt:     now,
		Data:          &replayData,
	}

	data, err := json.Marshal(replay)
	r.NoError(err)
	r.Contains(string(data), `"duration_ticks":250`)
	r.Contains(string(data), `"winner":"bot-1"`)
	r.Contains(string(data), `"events":["spawn:bot-1","spawn:bot-2"]`)

	var parsed Replay
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("replay-1", parsed.Id)
	r.NotNil(parsed.Data)
	r.Equal("bot-1", parsed.Data.Winner)
	r.Len(parsed.Data.Frames, 1)
	r.Equal(1, parsed.Data.Frames[0].Tick)
}
