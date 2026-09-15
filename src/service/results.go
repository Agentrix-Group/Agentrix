package service

import (
	"context"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/google/uuid"
)

func (s *service) ListResults(ctx context.Context) ([]model.Result, error) {
	return s.repo.ListResults(ctx)
}

func (s *service) ListResultsByMatch(ctx context.Context, matchId string) ([]model.Result, error) {
	return s.repo.ListResultsByMatch(ctx, matchId)
}

func (s *service) GetResult(ctx context.Context, id string) (*model.Result, error) {
	return s.repo.GetResult(ctx, id)
}

func (s *service) CreateResult(ctx context.Context, result *model.Result) error {
	if result.Id == "" {
		result.Id = uuid.New().String()
	}
	result.CreatedAt = time.Now().UTC()

	return s.repo.CreateResult(ctx, result)
}
