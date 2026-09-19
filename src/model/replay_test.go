package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReplayNDJSONRecordsJSON(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	replay := Replay{
		Id:            "replay-1",
		MatchId:       "match-100",
		FilePath:      "/storage/replays/match-100.ndjson",
		DurationTicks: 250,
		Summary:       "bot-1 eliminated bot-2",
		Active:        true,
		CreatedAt:     now,
	}

	data, err := json.Marshal(replay)
	r.NoError(err)
	r.Contains(string(data), `"duration_ticks":250`)

	var parsed Replay
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("replay-1", parsed.Id)

	metadata := ReplayMetadata{
		Type: "metadata", ReplayID: replay.Id, MatchID: replay.MatchId, GameID: "starfighter",
		Seed: 42, Participants: []string{"bot-1", "bot-2"}, FixedTimestepMs: 17, CreatedAt: now,
	}
	data, err = json.Marshal(metadata)
	r.NoError(err)
	r.Contains(string(data), `"game_id":"starfighter"`)
	r.Contains(string(data), `"fixed_timestep_ms":17`)
}
