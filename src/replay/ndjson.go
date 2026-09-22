package replay

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

type StreamWriter interface {
	WriteSnapshot(snapshot model.ReplaySnapshot) error
	Complete(result model.ReplayResult) error
	Close() error
	FrameCount() int
}

type ndjsonWriter struct {
	mu        sync.Mutex
	encoder   *json.Encoder
	closer    io.Closer
	frames    int
	lastHash  string
	completed bool
	closed    bool
}

func NewStreamWriter(destination io.WriteCloser, metadata model.ReplayMetadata) (StreamWriter, error) {
	if destination == nil {
		return nil, errors.New("replay destination is nil")
	}
	if !validMetadata(metadata) {
		return nil, errors.New("invalid Starfighter replay metadata")
	}
	metadata.Type = "metadata"
	writer := &ndjsonWriter{encoder: json.NewEncoder(destination), closer: destination}
	writer.encoder.SetEscapeHTML(false)
	if err := writer.encoder.Encode(metadata); err != nil {
		_ = destination.Close()
		return nil, err
	}
	return writer, nil
}

func (w *ndjsonWriter) WriteSnapshot(snapshot model.ReplaySnapshot) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.completed {
		return errors.New("replay stream is closed")
	}
	if !json.Valid(snapshot.PublicSnapshot) {
		return errors.New("public snapshot is not valid JSON")
	}
	var publicState struct {
		Tick      *int   `json:"tick"`
		StateHash string `json:"stateHash"`
	}
	if err := json.Unmarshal(snapshot.PublicSnapshot, &publicState); err != nil || publicState.Tick == nil {
		return errors.New("public snapshot must be a JSON object")
	}
	if snapshot.Tick != w.frames {
		return errors.New("replay snapshots must start at tick 0 and remain sequential")
	}
	if *publicState.Tick != snapshot.Tick {
		return errors.New("public snapshot tick does not match replay envelope")
	}
	if snapshot.StateHash == "" || (publicState.StateHash != "" && publicState.StateHash != snapshot.StateHash) {
		return errors.New("public snapshot state hash does not match replay envelope")
	}
	snapshot.Type = "snapshot"
	if err := w.encoder.Encode(snapshot); err != nil {
		return err
	}
	w.frames++
	w.lastHash = snapshot.StateHash
	return nil
}

func (w *ndjsonWriter) Complete(result model.ReplayResult) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return errors.New("replay stream is closed")
	}
	if w.completed {
		return errors.New("replay result already written")
	}
	if w.frames == 0 || result.FinalTick != w.frames-1 {
		return errors.New("replay result final tick does not match snapshots")
	}
	if result.FinalStateHash == "" || result.FinalStateHash != w.lastHash {
		return errors.New("replay result state hash does not match final snapshot")
	}
	if result.Scores == nil || result.FinishedAt.IsZero() {
		return errors.New("replay result scores and finished_at are required")
	}
	if !validResultReason(result.Reason) {
		return errors.New("invalid replay result reason")
	}
	result.Type = "result"
	if err := w.encoder.Encode(result); err != nil {
		return err
	}
	w.completed = true
	return nil
}

func (w *ndjsonWriter) FrameCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.frames
}

func (w *ndjsonWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}
	w.closed = true
	return w.closer.Close()
}

func DecodeNDJSON(source io.Reader) (*model.ReplayDocument, error) {
	scanner := bufio.NewScanner(source)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	document := &model.ReplayDocument{}
	seenMetadata := false
	seenResult := false
	for scanner.Scan() {
		line := scanner.Bytes()
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			return nil, err
		}
		switch envelope.Type {
		case "metadata":
			if seenMetadata || len(document.Snapshots) > 0 {
				return nil, errors.New("replay metadata must be the first line")
			}
			if err := json.Unmarshal(line, &document.Metadata); err != nil {
				return nil, err
			}
			if !validMetadata(document.Metadata) {
				return nil, errors.New("invalid Starfighter replay metadata")
			}
			seenMetadata = true
		case "snapshot":
			if !seenMetadata || seenResult {
				return nil, errors.New("snapshot outside replay body")
			}
			var snapshot model.ReplaySnapshot
			if err := json.Unmarshal(line, &snapshot); err != nil {
				return nil, err
			}
			var publicState struct {
				Tick      *int   `json:"tick"`
				StateHash string `json:"stateHash"`
			}
			if err := json.Unmarshal(snapshot.PublicSnapshot, &publicState); err != nil || publicState.Tick == nil ||
				snapshot.Tick != len(document.Snapshots) || *publicState.Tick != snapshot.Tick || snapshot.StateHash == "" ||
				(publicState.StateHash != "" && publicState.StateHash != snapshot.StateHash) {
				return nil, errors.New("invalid replay snapshot sequence")
			}
			document.Snapshots = append(document.Snapshots, snapshot)
		case "result":
			if !seenMetadata || seenResult {
				return nil, errors.New("invalid replay result line")
			}
			if err := json.Unmarshal(line, &document.Result); err != nil {
				return nil, err
			}
			if !validResultReason(document.Result.Reason) || document.Result.Scores == nil || document.Result.FinishedAt.IsZero() {
				return nil, errors.New("invalid replay result reason")
			}
			seenResult = true
		default:
			return nil, errors.New("unknown replay line type")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !seenMetadata || !seenResult || len(document.Snapshots) == 0 {
		return nil, errors.New("incomplete replay stream")
	}
	last := document.Snapshots[len(document.Snapshots)-1]
	if document.Result.FinalTick != last.Tick || document.Result.FinalStateHash != last.StateHash {
		return nil, errors.New("replay result does not seal the final snapshot")
	}
	return document, nil
}

func validResultReason(reason string) bool {
	switch reason {
	case "eliminated", "timeout", "score_limit":
		return true
	default:
		return false
	}
}

func validMetadata(metadata model.ReplayMetadata) bool {
	if metadata.MatchID == "" || metadata.ReplayID == "" || metadata.GameID != "starfighter" ||
		len(metadata.Participants) < model.StarfighterMinPlayers ||
		len(metadata.Participants) > model.StarfighterMaxPlayers ||
		metadata.FixedTimestepMs <= 0 || metadata.CreatedAt.IsZero() {
		return false
	}
	seen := make(map[string]bool, len(metadata.Participants))
	for _, participant := range metadata.Participants {
		if participant == "" || seen[participant] {
			return false
		}
		seen[participant] = true
	}
	return true
}
