package service

import (
	"context"
	"database/sql"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func (s *service) ListGames(ctx context.Context) ([]model.Game, error) {
	game, err := s.repo.GetGame(ctx, "starfighter")
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.Game{}, nil
		}
		return nil, err
	}
	return []model.Game{*game}, nil
}

func (s *service) GetGame(ctx context.Context, id string) (*model.Game, error) {
	if id != "starfighter" {
		return nil, sql.ErrNoRows
	}
	return s.repo.GetGame(ctx, id)
}
