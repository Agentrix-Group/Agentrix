package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	subpath := fmt.Sprintf("replays/tmp/%s.ndjson", replay.Id)
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
			// Check temporary uncommitted path
			tmpSubpath := fmt.Sprintf("replays/tmp/%s.ndjson", id)
			tmpZstSubpath := fmt.Sprintf("replays/tmp/%s.ndjson.zst", id)
			if s.artifacts.Exists(tmpSubpath) {
				tmpRaw, tmpErr := s.artifacts.Read(ctx, tmpSubpath)
				if tmpErr != nil {
					return nil, tmpErr
				}
				raw = tmpRaw
				subpath = tmpSubpath
			} else if s.artifacts.Exists(tmpZstSubpath) {
				decompressed, decErr := replaystream.DecompressZstd(ctx, s.artifacts.GetPath(tmpZstSubpath))
				if decErr != nil {
					return nil, decErr
				}
				raw = decompressed
				subpath = tmpZstSubpath
			} else {
				return nil, err
			}
		}
	}
	document, err := replaystream.DecodeNDJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(raw)
	return &model.Replay{
		Id:            id,
		MatchId:       document.Metadata.MatchID,
		FilePath:      s.artifacts.GetPath(subpath),
		DurationTicks: len(document.Snapshots),
		Summary:       fmt.Sprintf("Winner: %s, Frames: %d, Reason: %s", document.Result.Winner, len(document.Snapshots), document.Result.Reason),
		Sha256:        hex.EncodeToString(h[:]),
		SizeBytes:     int64(len(raw)),
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
	tmpSubpath := fmt.Sprintf("replays/tmp/%s.ndjson", id)
	tmpRaw, tmpErr := s.artifacts.Read(ctx, tmpSubpath)
	if tmpErr == nil {
		return tmpRaw, nil
	}
	tmpZstSubpath := fmt.Sprintf("replays/tmp/%s.ndjson.zst", id)
	if s.artifacts.Exists(tmpZstSubpath) {
		return replaystream.DecompressZstd(ctx, s.artifacts.GetPath(tmpZstSubpath))
	}
	return nil, err
}

func (s *service) PublishReplay(ctx context.Context, replayID string) (*model.Replay, error) {
	tmpSubpath := fmt.Sprintf("replays/tmp/%s.ndjson", replayID)
	finalSubpath := fmt.Sprintf("replays/%s.ndjson", replayID)

	tmpZst := fmt.Sprintf("replays/tmp/%s.ndjson.zst", replayID)
	finalZst := fmt.Sprintf("replays/%s.ndjson.zst", replayID)

	if s.artifacts.Exists(tmpSubpath) {
		if err := s.artifacts.Move(ctx, tmpSubpath, finalSubpath); err != nil {
			return nil, fmt.Errorf("failed to publish replay: %w", err)
		}
	}
	if s.artifacts.Exists(tmpZst) {
		if err := s.artifacts.Move(ctx, tmpZst, finalZst); err != nil {
			return nil, fmt.Errorf("failed to publish compressed replay: %w", err)
		}
	}

	return s.GetReplay(ctx, replayID)
}

func (s *service) DiscardReplay(ctx context.Context, replayID string) error {
	tmpSubpath := fmt.Sprintf("replays/tmp/%s.ndjson", replayID)
	tmpZst := fmt.Sprintf("replays/tmp/%s.ndjson.zst", replayID)
	_ = s.artifacts.Delete(ctx, tmpSubpath)
	_ = s.artifacts.Delete(ctx, tmpZst)
	return nil
}
