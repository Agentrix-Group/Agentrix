package repository

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestContestsRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListContests
	contests, err := repo.ListContests(ctx)
	r.Error(err)
	r.Nil(contests)

	// ListPublicContests
	publicContests, err := repo.ListPublicContests(ctx, model.PublicContestsFilter{})
	r.Error(err)
	r.Nil(publicContests)

	// GetContest
	contest, err := repo.GetContest(ctx, "c1")
	r.Error(err)
	r.Nil(contest)

	// CreateContest
	err = repo.CreateContest(ctx, &model.Contest{Name: "Tournament"})
	r.Error(err)

	// UpdateContest
	err = repo.UpdateContest(ctx, &model.Contest{Id: "c1", Name: "Tournament 2"})
	r.Error(err)

	// ActivateContest
	err = repo.ActivateContest(ctx, "c1", false)
	r.Error(err)

	// ListCategories
	categories, err := repo.ListCategories(ctx)
	r.Error(err)
	r.Nil(categories)

	// GetCategory
	cat, err := repo.GetCategory(ctx, "cat-1")
	r.Error(err)
	r.Nil(cat)

	// CreateCategory
	err = repo.CreateCategory(ctx, &model.Category{Description: "Strategy"})
	r.Error(err)

	// UpdateCategory
	err = repo.UpdateCategory(ctx, &model.Category{Id: "cat-1", Description: "Strategy 2"})
	r.Error(err)

	// ActivateCategory
	err = repo.ActivateCategory(ctx, "cat-1", false)
	r.Error(err)
}
