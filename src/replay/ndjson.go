// Package replay writes and validates the NDJSON replay stream
// (agentrix-replay/2): one metadata line, one snapshot line per simulated
// tick starting at 0, and one result line that seals the final state hash.
package replay

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

const FormatVersion = "agentrix-replay/2"

type TickRate struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

type Participant struct {
	PlayerID  string `json:"player_id"`
	SlotIndex int    `json:"slot_index"`
}

type Metadata struct {
	Type          string        `json:"type"`
	Format        string        `json:"format"`
	ReplayID      string        `json:"replay_id"`
	MatchID       string        `json:"match_id"`
	RunID         string        `json:"run_id"`
	SpecHash      string        `json:"spec_hash"`
	GameID        string        `json:"game_id"`
	GameVersion   string        `json:"game_version"`
	ConfigHash    string        `json:"config_hash"`
	EngineVersion string        `json:"engine_version"`
	EngineSHA256  string        `json:"engine_sha256"`
	Seed          int64         `json:"seed"`
	TickRate      TickRate      `json:"tick_rate"`
	Participants  []Participant `json:"participants"`
	CreatedAt     time.Time     `json:"created_at"`
}

type Snapshot struct {
	Type           string          `json:"type"`
	Tick           int             `json:"tick"`
	StateHash      string          `json:"state_hash"`
	Events         []string        `json:"events"`
	PublicSnapshot json.RawMessage `json:"public_snapshot"`
}

type Result struct {
	Type           string         `json:"type"`
	FinalTick      int            `json:"final_tick"`
	Winner         string         `json:"winner"`
	Scores         map[string]int `json:"scores"`
	Reason         string         `json:"reason"`
	FinalStateHash string         `json:"final_state_hash"`
	FinishedAt     time.Time      `json:"finished_at"`
}

type Document struct {
	Metadata  Metadata
	Snapshots []Snapshot
	Result    Result
}

func (m Metadata) validate() error {
	switch {
	case m.ReplayID == "" || m.MatchID == "" || m.RunID == "" || m.GameID == "":
		return errors.New("replay metadata requires replay, match, run and game identifiers")
	case m.SpecHash == "" || m.EngineSHA256 == "" || m.ConfigHash == "":
		return errors.New("replay metadata requires spec, engine and config digests")
	case m.TickRate.Numerator <= 0 || m.TickRate.Denominator <= 0:
		return errors.New("replay metadata requires a positive tick_rate")
	case len(m.Participants) == 0 || m.CreatedAt.IsZero():
		return errors.New("replay metadata requires participants and created_at")
	}
	seen := map[string]bool{}
	for i, p := range m.Participants {
		if p.PlayerID == "" || p.SlotIndex != i || seen[p.PlayerID] {
			return fmt.Errorf("invalid replay participant %d", i)
		}
		seen[p.PlayerID] = true
	}
	return nil
}

// Writer streams a replay. Every method returns its error; nothing is
// silently skipped.
type Writer struct {
	mu        sync.Mutex
	out       *bufio.Writer
	encoder   *json.Encoder
	frames    int
	lastHash  string
	completed bool
}

func NewWriter(destination io.Writer, metadata Metadata) (*Writer, error) {
	metadata.Type, metadata.Format = "metadata", FormatVersion
	if err := metadata.validate(); err != nil {
		return nil, err
	}
	buffered := bufio.NewWriterSize(destination, 64*1024)
	w := &Writer{out: buffered, encoder: json.NewEncoder(buffered)}
	w.encoder.SetEscapeHTML(false)
	if err := w.encoder.Encode(metadata); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *Writer) WriteSnapshot(s Snapshot) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.completed {
		return errors.New("replay already sealed")
	}
	if s.Tick != w.frames {
		return fmt.Errorf("replay snapshot tick %d out of sequence (expected %d)", s.Tick, w.frames)
	}
	if s.StateHash == "" || !json.Valid(s.PublicSnapshot) {
		return errors.New("replay snapshot requires a state hash and a JSON public snapshot")
	}
	if s.Events == nil {
		s.Events = []string{}
	}
	s.Type = "snapshot"
	if err := w.encoder.Encode(s); err != nil {
		return err
	}
	w.frames++
	w.lastHash = s.StateHash
	return nil
}

// Complete writes the result line and flushes the stream.
func (w *Writer) Complete(r Result) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.completed {
		return errors.New("replay already sealed")
	}
	if w.frames == 0 || r.FinalTick != w.frames-1 || r.FinalStateHash != w.lastHash {
		return errors.New("replay result does not seal the final snapshot")
	}
	if r.Scores == nil || r.Reason == "" || r.FinishedAt.IsZero() {
		return errors.New("replay result requires scores, reason and finished_at")
	}
	r.Type = "result"
	if err := w.encoder.Encode(r); err != nil {
		return err
	}
	w.completed = true
	return w.out.Flush()
}

func (w *Writer) FrameCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.frames
}

// Decode validates a complete replay stream.
func Decode(source io.Reader) (*Document, error) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)
	doc := &Document{}
	state := 0 // 0 metadata expected, 1 snapshots, 2 sealed
	for scanner.Scan() {
		line := scanner.Bytes()
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			return nil, err
		}
		switch {
		case envelope.Type == "metadata" && state == 0:
			if err := json.Unmarshal(line, &doc.Metadata); err != nil {
				return nil, err
			}
			if doc.Metadata.Format != FormatVersion {
				return nil, fmt.Errorf("unsupported replay format %q", doc.Metadata.Format)
			}
			if err := doc.Metadata.validate(); err != nil {
				return nil, err
			}
			state = 1
		case envelope.Type == "snapshot" && state == 1:
			var s Snapshot
			if err := json.Unmarshal(line, &s); err != nil {
				return nil, err
			}
			if s.Tick != len(doc.Snapshots) || s.StateHash == "" {
				return nil, errors.New("invalid replay snapshot sequence")
			}
			doc.Snapshots = append(doc.Snapshots, s)
		case envelope.Type == "result" && state == 1:
			if err := json.Unmarshal(line, &doc.Result); err != nil {
				return nil, err
			}
			state = 2
		default:
			return nil, fmt.Errorf("unexpected replay line %q", envelope.Type)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if state != 2 || len(doc.Snapshots) == 0 {
		return nil, errors.New("incomplete replay stream")
	}
	last := doc.Snapshots[len(doc.Snapshots)-1]
	if doc.Result.FinalTick != last.Tick || doc.Result.FinalStateHash != last.StateHash {
		return nil, errors.New("replay result does not seal the final snapshot")
	}
	return doc, nil
}
