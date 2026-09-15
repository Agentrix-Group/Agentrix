package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/google/uuid"
)

func (s *service) GetReplay(ctx context.Context, id string) (*model.Replay, error) {

	subpath := fmt.Sprintf("replays/%s.json", id)
	dataBytes, err := s.artifacts.Read(ctx, subpath)
	if err != nil {
		return nil, err
	}

	var replayData model.ReplayData
	if err := json.Unmarshal(dataBytes, &replayData); err != nil {
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
		return err
	}

	filePath, err := s.artifacts.Save(ctx, subpath, dataBytes)
	if err != nil {
		return err
	}

	replay.FilePath = filePath
	replay.DurationTicks = len(data.Frames)
	replay.Active = true
	replay.CreatedAt = time.Now().UTC()

	return nil
}

func (s *service) StreamReplay(ctx context.Context, id string) ([]byte, error) {
	subpath := fmt.Sprintf("replays/%s.json", id)
	return s.artifacts.Read(ctx, subpath)
}
