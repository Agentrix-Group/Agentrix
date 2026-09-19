package repository

import (
	"context"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestMatchCommitterAndRunRepository_Disconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// Verify repo implements MatchCommitter and MatchRunRepository
	committer, ok := repo.(MatchCommitter)
	r.True(ok, "repository must implement MatchCommitter")

	runRepo, ok := repo.(MatchRunRepository)
	r.True(ok, "repository must implement MatchRunRepository")

	// Verify calls fail cleanly when disconnected
	err := committer.CommitMatchResult(ctx, model.MatchResultCommit{
		MatchID:      "m-1",
		FencingToken: 1,
	})
	r.Error(err)

	err = runRepo.CreateMatchRun(ctx, &model.MatchRun{
		Id:           "run-1",
		MatchId:      "m-1",
		WorkerId:     "w-1",
		FencingToken: 1,
		Status:       model.MatchRunStatusRunning,
		StartedAt:    time.Now().UTC(),
	})
	r.Error(err)

	run, err := runRepo.GetMatchRun(ctx, "run-1")
	r.Error(err)
	r.Nil(run)

	err = runRepo.UpdateMatchRunHeartbeat(ctx, "run-1", time.Now().UTC())
	r.Error(err)
}
