package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/config"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestReplaysService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	store, err := connection.NewArtifactStore(ctx, &config.Config{
		Artifacts: config.Artifacts{Dir: tempDir},
	})
	r.NoError(err)

	svc := NewService(nil, store, nil)

	replay := &model.Replay{
		Id:      "rep-123",
		MatchId: "match-123",
	}

	writer, err := svc.OpenReplay(ctx, replay, model.ReplayMetadata{
		ReplayID: "rep-123", MatchID: "match-123", GameID: "starfighter", Seed: 42,
		Participants: []string{"bot-1", "bot-2"}, FixedTimestepMs: 17, CreatedAt: time.Now().UTC(),
	})
	r.NoError(err)
	snapshot, _ := json.Marshal(map[string]any{"tick": 0, "fighters": []any{}, "bullets": []any{}, "stateHash": "hash-0"})
	r.NoError(writer.WriteSnapshot(model.ReplaySnapshot{Tick: 0, PublicSnapshot: snapshot, StateHash: "hash-0"}))
	r.NoError(writer.Complete(model.ReplayResult{
		FinalTick: 0, Winner: "bot-1", Scores: map[string]int{"bot-1": 100, "bot-2": 0},
		Reason: "eliminated", FinalStateHash: "hash-0", FinishedAt: time.Now().UTC(),
	}))
	r.NoError(writer.Close())
	r.NotEmpty(replay.FilePath)

	// GetReplay
	fetched, err := svc.GetReplay(ctx, "rep-123")
	r.NoError(err)
	r.NotNil(fetched)
	r.Equal("rep-123", fetched.Id)
	r.Equal(1, fetched.DurationTicks)
	r.Contains(fetched.Summary, "bot-1")

	// StreamReplay
	raw, err := svc.StreamReplay(ctx, "rep-123")
	r.NoError(err)
	r.NotEmpty(raw)
	r.Contains(string(raw), "\"type\":\"metadata\"")
	r.Contains(string(raw), "starfighter")
	r.Contains(string(raw), "\"type\":\"result\"")
}
