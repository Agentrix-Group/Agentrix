package service

import (
	"context"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/google/uuid"
)

func (s *service) ListGames(ctx context.Context) ([]model.Game, error) {
	return s.repo.ListGames(ctx)
}

func (s *service) GetGame(ctx context.Context, id string) (*model.Game, error) {
	return s.repo.GetGame(ctx, id)
}

func (s *service) CreateGame(ctx context.Context, game *model.Game) error {
	if game.Id == "" {
		game.Id = uuid.New().String()
	}
	if game.MinPlayers <= 0 {
		game.MinPlayers = 2
	}
	if game.MaxPlayers <= 0 {
		game.MaxPlayers = 4
	}
	game.Active = true
	game.CreatedAt = time.Now().UTC()

	return s.repo.CreateGame(ctx, game)
}

func (s *service) UpdateGame(ctx context.Context, game *model.Game) error {
	return s.repo.UpdateGame(ctx, game)
}

func (s *service) ActivateGame(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateGame(ctx, id, isActive)
}
