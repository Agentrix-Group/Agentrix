package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/model"
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

func TestReplay_AtomicPublishAndDiscard(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	tempDir := t.TempDir()
	store, err := connection.NewArtifactStore(ctx, &config.Config{
		Artifacts: config.Artifacts{Dir: tempDir},
	})
	r.NoError(err)

	svc := NewService(nil, store, nil)

	// 1. Open and write replay -> starts in replays/tmp/
	replay := &model.Replay{Id: "rep-pub-1", MatchId: "match-pub-1"}
	writer, err := svc.OpenReplay(ctx, replay, model.ReplayMetadata{
		ReplayID: "rep-pub-1", MatchID: "match-pub-1", GameID: "starfighter", Seed: 99,
		Participants: []string{"p1", "p2"}, FixedTimestepMs: 17, CreatedAt: time.Now().UTC(),
	})
	r.NoError(err)
	snapshot, _ := json.Marshal(map[string]any{"tick": 0, "fighters": []any{}, "bullets": []any{}, "stateHash": "h0"})
	r.NoError(writer.WriteSnapshot(model.ReplaySnapshot{Tick: 0, PublicSnapshot: snapshot, StateHash: "h0"}))
	r.NoError(writer.Complete(model.ReplayResult{
		FinalTick: 0, Winner: "p1", Scores: map[string]int{"p1": 50, "p2": 0},
		Reason: "score_limit", FinalStateHash: "h0", FinishedAt: time.Now().UTC(),
	}))
	r.NoError(writer.Close())

	// Verify temporary file exists, canonical does not exist yet
	r.True(store.Exists("replays/tmp/rep-pub-1.ndjson"))
	r.False(store.Exists("replays/rep-pub-1.ndjson"))

	// Publish
	published, err := svc.PublishReplay(ctx, "rep-pub-1")
	r.NoError(err)
	r.NotNil(published)
	r.NotEmpty(published.Sha256)
	r.Greater(published.SizeBytes, int64(0))

	// Verify canonical exists, temporary no longer exists
	r.True(store.Exists("replays/rep-pub-1.ndjson"))
	r.False(store.Exists("replays/tmp/rep-pub-1.ndjson"))

	// 2. Discard temporary replay
	replay2 := &model.Replay{Id: "rep-discard", MatchId: "match-discard"}
	writer2, err := svc.OpenReplay(ctx, replay2, model.ReplayMetadata{
		ReplayID: "rep-discard", MatchID: "match-discard", GameID: "starfighter", Seed: 1,
		Participants: []string{"p1", "p2"}, FixedTimestepMs: 17, CreatedAt: time.Now().UTC(),
	})
	r.NoError(err)
	_ = writer2.Close()
	r.True(store.Exists("replays/tmp/rep-discard.ndjson"))

	r.NoError(svc.DiscardReplay(ctx, "rep-discard"))
	r.False(store.Exists("replays/tmp/rep-discard.ndjson"))
	r.False(store.Exists("replays/rep-discard.ndjson"))
}
