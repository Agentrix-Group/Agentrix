package repository

import (
	"context"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestResultsRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListResults
	results, err := repo.ListResults(ctx)
	r.Error(err)
	r.Nil(results)

	// ListResultsByMatch
	matchResults, err := repo.ListResultsByMatch(ctx, "m1")
	r.Error(err)
	r.Nil(matchResults)

	// GetResult
	res, err := repo.GetResult(ctx, "res-1")
	r.Error(err)
	r.Nil(res)

	// CreateResult
	err = repo.CreateResult(ctx, &model.Result{Id: "res-1", Score: 100})
	r.Error(err)
}
