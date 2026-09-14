package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
	"github.com/google/uuid"
)

func (s *service) GetReplay(ctx context.Context, id string) (*model.Replay, error) {
	tracer.Debugf(ctx, "Retrieving replay metadata and data for %s", id)

	subpath := fmt.Sprintf("replays/%s.json", id)
	dataBytes, err := s.artifacts.Read(ctx, subpath)
	if err != nil {
		tracer.Warnf(ctx, "Replay artifact not found: %s", subpath)
		return nil, err
	}

	var replayData model.ReplayData
	if err := json.Unmarshal(dataBytes, &replayData); err != nil {
		tracer.Errorf(ctx, "Failed to unmarshal replay data: %s", err)
		return nil, err
	}

	replay := &model.Replay{
		Id:            id,
		MatchId:       replayData.MatchId,
		FilePath:      s.artifacts.GetPath(subpath),
		DurationTicks: len(replayData.Frames),
		Summary:       fmt.Sprintf("Winner: %s, Frames: %d", replayData.Winner, len(replayData.Frames)),
		Active:        true,
		CreatedAt:     time.Now().UTC(),
		Data:          &replayData,
	}

	return replay, nil
}

func (s *service) SaveReplay(ctx context.Context, replay *model.Replay, data *model.ReplayData) error {
	if replay.Id == "" {
		replay.Id = uuid.New().String()
	}

	subpath := fmt.Sprintf("replays/%s.json", replay.Id)
	dataBytes, err := json.Marshal(data)
	if err != nil {
		tracer.Errorf(ctx, "Failed to serialize replay data: %s", err)
		return err
	}

	filePath, err := s.artifacts.Save(ctx, subpath, dataBytes)
	if err != nil {
		tracer.Errorf(ctx, "Failed to write replay artifact: %s", err)
		return err
	}

	replay.FilePath = filePath
	replay.DurationTicks = len(data.Frames)
	replay.Active = true
	replay.CreatedAt = time.Now().UTC()

	tracer.Debugf(ctx, "Successfully saved replay %s (%d ticks)", replay.Id, replay.DurationTicks)
	return nil
}

func (s *service) StreamReplay(ctx context.Context, id string) ([]byte, error) {
	tracer.Debugf(ctx, "Streaming replay bytes for %s", id)
	subpath := fmt.Sprintf("replays/%s.json", id)
	return s.artifacts.Read(ctx, subpath)
}
