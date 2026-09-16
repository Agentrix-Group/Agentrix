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
	listGamesFn    func(ctx context.Context) ([]model.Game, error)
	getGameFn      func(ctx context.Context, id string) (*model.Game, error)
	createGameFn   func(ctx context.Context, game *model.Game) error
	updateGameFn   func(ctx context.Context, game *model.Game) error
	activateGameFn func(ctx context.Context, id string, isActive bool) error
}

func (m *mockGamesRepo) ListGames(ctx context.Context) ([]model.Game, error) {
	if m.listGamesFn != nil {
		return m.listGamesFn(ctx)
	}
	return nil, nil
}

func (m *mockGamesRepo) GetGame(ctx context.Context, id string) (*model.Game, error) {
	if m.getGameFn != nil {
		return m.getGameFn(ctx, id)
	}
	return &model.Game{Id: id, Name: "TestGame"}, nil
}

func (m *mockGamesRepo) CreateGame(ctx context.Context, game *model.Game) error {
	if m.createGameFn != nil {
		return m.createGameFn(ctx, game)
	}
	return nil
}

func (m *mockGamesRepo) UpdateGame(ctx context.Context, game *model.Game) error {
	if m.updateGameFn != nil {
		return m.updateGameFn(ctx, game)
	}
	return nil
}

func (m *mockGamesRepo) ActivateGame(ctx context.Context, id string, isActive bool) error {
	if m.activateGameFn != nil {
		return m.activateGameFn(ctx, id, isActive)
	}
	return nil
}

func TestGamesService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	createdGames := make([]*model.Game, 0)
	mockRepo := &mockGamesRepo{
		listGamesFn: func(ctx context.Context) ([]model.Game, error) {
			return []model.Game{{Id: "arena-basica", Name: "Arena"}}, nil
		},
		createGameFn: func(ctx context.Context, g *model.Game) error {
			createdGames = append(createdGames, g)
			return nil
		},
	}

	svc := NewService(mockRepo, nil, nil)

	// ListGames
	games, err := svc.ListGames(ctx)
	r.NoError(err)
	r.Len(games, 1)

	// GetGame
	game, err := svc.GetGame(ctx, "arena-basica")
	r.NoError(err)
	r.Equal("arena-basica", game.Id)

	// CreateGame
	newGame := &model.Game{Name: "CustomArena"}
	err = svc.CreateGame(ctx, newGame)
	r.NoError(err)
	r.NotEmpty(newGame.Id)
	r.Equal(2, newGame.MinPlayers) // defaults applied
	r.Equal(4, newGame.MaxPlayers)
	r.True(newGame.Active)
	r.Len(createdGames, 1)

	// UpdateGame & ActivateGame
	r.NoError(svc.UpdateGame(ctx, newGame))
	r.NoError(svc.ActivateGame(ctx, newGame.Id, false))
}
