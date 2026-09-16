package repository

import (
	"context"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestGamesRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListGames
	games, err := repo.ListGames(ctx)
	r.Error(err)
	r.Nil(games)

	// GetGame
	game, err := repo.GetGame(ctx, "arena-basica")
	r.Error(err)
	r.Nil(game)

	// CreateGame
	err = repo.CreateGame(ctx, &model.Game{Name: "Arena"})
	r.Error(err)

	// UpdateGame
	err = repo.UpdateGame(ctx, &model.Game{Id: "arena-basica", Name: "Arena 2"})
	r.Error(err)

	// ActivateGame
	err = repo.ActivateGame(ctx, "arena-basica", false)
	r.Error(err)
}
