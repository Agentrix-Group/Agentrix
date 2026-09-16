package service

import (
	"context"
	"testing"

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

	data := &model.ReplayData{
		GameId:   "arena-basica",
		MatchId:  "match-123",
		Seed:     42,
		Winner:   "bot-1",
		Scores:   map[string]int{"bot-1": 100},
		Players:  []string{"bot-1"},
		MaxTicks: 50,
		Frames: []model.ReplayFrame{
			{Tick: 1, Events: []string{"start"}},
		},
	}

	// SaveReplay
	err = svc.SaveReplay(ctx, replay, data)
	r.NoError(err)
	r.NotEmpty(replay.FilePath)
	r.Equal(1, replay.DurationTicks)

	// GetReplay
	fetched, err := svc.GetReplay(ctx, "rep-123")
	r.NoError(err)
	r.NotNil(fetched)
	r.Equal("rep-123", fetched.Id)
	r.Equal("bot-1", fetched.Data.Winner)

	// StreamReplay
	raw, err := svc.StreamReplay(ctx, "rep-123")
	r.NoError(err)
	r.NotEmpty(raw)
	r.Contains(string(raw), "arena-basica")
}
