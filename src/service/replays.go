package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	replaystream "github.com/F4nk1/Agentrix/src/replay"
	"github.com/google/uuid"
)

func (s *service) OpenReplay(ctx context.Context, replay *model.Replay, metadata model.ReplayMetadata) (replaystream.StreamWriter, error) {
	if replay.Id == "" {
		replay.Id = uuid.New().String()
	}
	metadata.ReplayID = replay.Id
	metadata.Type = "metadata"
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now().UTC()
	}
	subpath := fmt.Sprintf("replays/%s.ndjson", replay.Id)
	destination, path, err := s.artifacts.OpenWriter(ctx, subpath)
	if err != nil {
		return nil, err
	}
	writer, err := replaystream.NewStreamWriter(destination, metadata)
	if err != nil {
		return nil, err
	}
	replay.MatchId = metadata.MatchID
	replay.FilePath = path
	replay.Active = true
	replay.CreatedAt = metadata.CreatedAt
	return writer, nil
}

func (s *service) GetReplay(ctx context.Context, id string) (*model.Replay, error) {
	subpath := fmt.Sprintf("replays/%s.ndjson", id)
	raw, err := s.artifacts.Read(ctx, subpath)
	if err != nil {
		zstSubpath := fmt.Sprintf("replays/%s.ndjson.zst", id)
		if s.artifacts.Exists(zstSubpath) {
			decompressed, decErr := replaystream.DecompressZstd(ctx, s.artifacts.GetPath(zstSubpath))
			if decErr != nil {
				return nil, decErr
			}
			raw = decompressed
			subpath = zstSubpath
		} else {
			return nil, err
		}
	}
	document, err := replaystream.DecodeNDJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	return &model.Replay{
		Id:            id,
		MatchId:       document.Metadata.MatchID,
		FilePath:      s.artifacts.GetPath(subpath),
		DurationTicks: len(document.Snapshots),
		Summary:       fmt.Sprintf("Winner: %s, Frames: %d, Reason: %s", document.Result.Winner, len(document.Snapshots), document.Result.Reason),
		Active:        true,
		CreatedAt:     document.Metadata.CreatedAt,
	}, nil
}

func (s *service) StreamReplay(ctx context.Context, id string) ([]byte, error) {
	subpath := fmt.Sprintf("replays/%s.ndjson", id)
	raw, err := s.artifacts.Read(ctx, subpath)
	if err == nil {
		return raw, nil
	}
	zstSubpath := fmt.Sprintf("replays/%s.ndjson.zst", id)
	if s.artifacts.Exists(zstSubpath) {
		return replaystream.DecompressZstd(ctx, s.artifacts.GetPath(zstSubpath))
	}
	return nil, err
}
