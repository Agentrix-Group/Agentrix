package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGamesRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// GetGame
	game, err := repo.GetGame(ctx, "starfighter")
	r.Error(err)
	r.Nil(game)
}
