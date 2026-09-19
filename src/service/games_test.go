package service

import (
	"context"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockGamesRepo struct {
	repository.Repository
	getGameFn func(ctx context.Context, id string) (*model.Game, error)
}

func (m *mockGamesRepo) GetGame(ctx context.Context, id string) (*model.Game, error) {
	if m.getGameFn != nil {
		return m.getGameFn(ctx, id)
	}
	return &model.Game{Id: id, Name: "TestGame"}, nil
}

func TestGamesService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	mockRepo := &mockGamesRepo{
		getGameFn: func(ctx context.Context, id string) (*model.Game, error) {
			return &model.Game{Id: id, Name: "Starfighter Arena", MinPlayers: 2, MaxPlayers: 2}, nil
		},
	}

	svc := NewService(mockRepo, nil, nil)

	// ListGames
	games, err := svc.ListGames(ctx)
	r.NoError(err)
	r.Len(games, 1)

	// GetGame
	game, err := svc.GetGame(ctx, "starfighter")
	r.NoError(err)
	r.Equal("starfighter", game.Id)

	_, err = svc.GetGame(ctx, "other-game")
	r.Error(err)
}
