package repository

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestSubmissionsRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListSubmissions
	submissions, err := repo.ListSubmissions(ctx)
	r.Error(err)
	r.Nil(submissions)

	// ListSubmissionsByAgent
	agentSubs, err := repo.ListSubmissionsByAgent(ctx, "agent-1")
	r.Error(err)
	r.Nil(agentSubs)

	// GetSubmission
	sub, err := repo.GetSubmission(ctx, "sub-1")
	r.Error(err)
	r.Nil(sub)

	// CreateSubmission
	err = repo.CreateSubmission(ctx, &model.Submission{AgentId: "agent-1", Version: 1})
	r.Error(err)
}
