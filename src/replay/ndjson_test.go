package replay

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestStreamWriterProducesSealedSequentialReplay(t *testing.T) {
	r := require.New(t)
	file, err := os.CreateTemp(t.TempDir(), "replay-*.ndjson")
	r.NoError(err)

	writer, err := NewStreamWriter(file, model.ReplayMetadata{
		ReplayID: "rep-1", MatchID: "match-1", GameID: "starfighter", Seed: 42,
		Participants: []string{"bot-1", "bot-2"}, FixedTimestepMs: 17, CreatedAt: time.Now().UTC(),
	})
	r.NoError(err)
	state, err := json.Marshal(map[string]any{"tick": 0, "fighters": []any{}, "bullets": []any{}, "stateHash": "hash-0"})
	r.NoError(err)
	r.NoError(writer.WriteSnapshot(model.ReplaySnapshot{Tick: 0, PublicSnapshot: state, StateHash: "hash-0"}))
	r.Error(writer.WriteSnapshot(model.ReplaySnapshot{Tick: 2, PublicSnapshot: state, StateHash: "hash-2"}))
	r.NoError(writer.Complete(model.ReplayResult{
		FinalTick: 0, Winner: "bot-1", Scores: map[string]int{"bot-1": 1, "bot-2": 0},
		Reason: "eliminated", FinalStateHash: "hash-0", FinishedAt: time.Now().UTC(),
	}))
	r.NoError(writer.Close())

	input, err := os.Open(file.Name())
	r.NoError(err)
	defer input.Close()
	document, err := DecodeNDJSON(input)
	r.NoError(err)
	r.Equal("match-1", document.Metadata.MatchID)
	r.Len(document.Snapshots, 1)
	r.Equal("hash-0", document.Result.FinalStateHash)
}

func TestStreamWriterRejectsNonCompetitiveResultReason(t *testing.T) {
	r := require.New(t)
	file, err := os.CreateTemp(t.TempDir(), "replay-*.ndjson")
	r.NoError(err)

	writer, err := NewStreamWriter(file, model.ReplayMetadata{
		ReplayID: "rep-2", MatchID: "match-2", GameID: "starfighter", Seed: 7,
		Participants: []string{"bot-1", "bot-2"}, FixedTimestepMs: 17, CreatedAt: time.Now().UTC(),
	})
	r.NoError(err)
	state := json.RawMessage(`{"tick":0,"fighters":[],"bullets":[],"stateHash":"hash-0"}`)
	r.NoError(writer.WriteSnapshot(model.ReplaySnapshot{Tick: 0, PublicSnapshot: state, StateHash: "hash-0"}))
	r.Error(writer.Complete(model.ReplayResult{
		FinalTick: 0, Scores: map[string]int{"bot-1": 0, "bot-2": 0},
		Reason: "execution_error", FinalStateHash: "hash-0", FinishedAt: time.Now().UTC(),
	}))
	r.NoError(writer.Close())
}
