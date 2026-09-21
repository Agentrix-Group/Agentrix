package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

// PublishPendingReplays moves staged replays of committed runs to their
// published location, verifying the digest before and after the move. A
// failure is recorded (publish_failed) and retried with backoff; the replay
// is never exposed before its artifact is verified.
func (s *Service) PublishPendingReplays(ctx context.Context, limit int) (published int, err error) {
	err = s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		pending, err := q.LockPendingReplays(ctx, now, limit)
		if err != nil {
			return err
		}
		for _, r := range pending {
			if pubErr := s.publishReplayFile(r); pubErr != nil {
				backoff := time.Duration(1<<min(r.PublishAttempts, 8)) * time.Second
				if err := q.MarkReplayPublishFailed(ctx, r.ID, pubErr.Error(), now.Add(backoff), now); err != nil {
					return err
				}
				continue
			}
			if err := q.MarkReplayPublished(ctx, r.ID, now); err != nil {
				return err
			}
			published++
		}
		return nil
	})
	return published, err
}

func (s *Service) publishReplayFile(r model.Replay) error {
	exists, err := s.artifacts.Exists(r.StorageKey)
	if err != nil {
		return err
	}
	if exists {
		// A previous attempt moved the file but crashed before marking it.
		return s.artifacts.Verify(r.StorageKey, r.SHA256)
	}
	if err := s.artifacts.Verify(r.StagingKey, r.SHA256); err != nil {
		if errors.Is(err, connection.ErrArtifactNotFound) {
			return fmt.Errorf("staged replay is missing")
		}
		return err
	}
	if err := s.artifacts.Move(r.StagingKey, r.StorageKey); err != nil {
		return err
	}
	return s.artifacts.Verify(r.StorageKey, r.SHA256)
}

// GetReplay returns a published replay visible to the principal.
func (s *Service) GetReplay(ctx context.Context, p model.Principal, id string) (*model.Replay, error) {
	if err := requireCap(p, model.CapReplaysView); err != nil {
		return nil, err
	}
	replay, err := s.store.GetReplay(ctx, id)
	if err != nil {
		return nil, err
	}
	if replay.Publication != model.ReplayPublished {
		return nil, model.NotFound("replay_not_found", "replay not found")
	}
	if _, err := s.visibleMatch(ctx, p, replay.MatchID); err != nil {
		return nil, model.NotFound("replay_not_found", "replay not found")
	}
	return replay, nil
}

// OpenReplay opens the verified published artifact for streaming.
func (s *Service) OpenReplay(ctx context.Context, p model.Principal, id string) (*model.Replay, io.ReadCloser, error) {
	replay, err := s.GetReplay(ctx, p, id)
	if err != nil {
		return nil, nil, err
	}
	f, err := s.artifacts.Open(replay.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("published replay %s is unreadable: %w", id, err)
	}
	return replay, f, nil
}
