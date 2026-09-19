package repository

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestRankingsRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListRankings
	rankings, err := repo.ListRankings(ctx)
	r.Error(err)
	r.Nil(rankings)

	// ListRankingsByContest
	contestRankings, err := repo.ListRankingsByContest(ctx, "c1")
	r.Error(err)
	r.Nil(contestRankings)

	// GetRanking
	ranking, err := repo.GetRanking(ctx, "r1")
	r.Error(err)
	r.Nil(ranking)

	// UpsertRanking
	err = repo.UpsertRanking(ctx, &model.Ranking{Id: "r1", Score: 100})
	r.Error(err)
}
