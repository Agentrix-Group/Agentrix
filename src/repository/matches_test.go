package repository

import (
	"context"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestMatchesRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListMatches
	matches, err := repo.ListMatches(ctx)
	r.Error(err)
	r.Nil(matches)

	// ListMatchesByContest
	contestMatches, err := repo.ListMatchesByContest(ctx, "c1")
	r.Error(err)
	r.Nil(contestMatches)

	// GetMatch
	match, err := repo.GetMatch(ctx, "m1")
	r.Error(err)
	r.Nil(match)

	// CreateMatch
	err = repo.CreateMatch(ctx, &model.Match{GameId: "starfighter"})
	r.Error(err)

	// UpdateMatch
	err = repo.UpdateMatch(ctx, &model.Match{Id: "m1", GameId: "starfighter"})
	r.Error(err)

	// ActivateMatch
	err = repo.ActivateMatch(ctx, "m1", false)
	r.Error(err)
}
