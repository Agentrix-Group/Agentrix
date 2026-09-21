package replay

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func metadata() Metadata {
	return Metadata{ReplayID: "r", MatchID: "m", RunID: "run", SpecHash: "s", GameID: "g", GameVersion: "1.0.0", ConfigHash: "c",
		EngineVersion: "1", EngineSHA256: "e", Seed: 1, TickRate: TickRate{60, 1},
		Participants: []Participant{{PlayerID: "a", SlotIndex: 0}, {PlayerID: "b", SlotIndex: 1}}, CreatedAt: time.Now()}
}

func snapshot(tick int, hash string) Snapshot {
	return Snapshot{Tick: tick, StateHash: hash, PublicSnapshot: json.RawMessage(`{"tick":` + string(rune('0'+tick)) + `}`)}
}

func TestWriterSealsSequentialReplay(t *testing.T) {
	var buf bytes.Buffer
	w, err := NewWriter(&buf, metadata())
	require.NoError(t, err)
	require.NoError(t, w.WriteSnapshot(snapshot(0, "h0")))
	require.Error(t, w.WriteSnapshot(snapshot(2, "h2")), "gaps are rejected")
	require.NoError(t, w.WriteSnapshot(snapshot(1, "h1")))
	require.Error(t, w.Complete(Result{FinalTick: 1, FinalStateHash: "other", Scores: map[string]int{}, Reason: "x", FinishedAt: time.Now()}))
	require.NoError(t, w.Complete(Result{FinalTick: 1, FinalStateHash: "h1", Scores: map[string]int{"a": 1}, Reason: "eliminated", FinishedAt: time.Now()}))
	require.Error(t, w.WriteSnapshot(snapshot(2, "h2")), "sealed replays are closed")
	doc, err := Decode(&buf)
	require.NoError(t, err)
	require.Len(t, doc.Snapshots, 2)
	require.Equal(t, FormatVersion, doc.Metadata.Format)
}

func TestMetadataRequiresDigestsAndTickRate(t *testing.T) {
	m := metadata()
	m.TickRate = TickRate{}
	_, err := NewWriter(&bytes.Buffer{}, m)
	require.Error(t, err)
	m = metadata()
	m.SpecHash = ""
	_, err = NewWriter(&bytes.Buffer{}, m)
	require.Error(t, err)
	m = metadata()
	m.Participants[1].PlayerID = "a"
	_, err = NewWriter(&bytes.Buffer{}, m)
	require.Error(t, err)
}

func TestDecodeRejectsTruncatedOrForeignStreams(t *testing.T) {
	var buf bytes.Buffer
	w, err := NewWriter(&buf, metadata())
	require.NoError(t, err)
	require.NoError(t, w.WriteSnapshot(snapshot(0, "h0")))
	_, err = Decode(bytes.NewReader(buf.Bytes()))
	require.Error(t, err, "unsealed stream")
	_, err = Decode(bytes.NewBufferString(`{"type":"metadata","format":"agentrix-replay/1"}` + "\n"))
	require.Error(t, err)
}
