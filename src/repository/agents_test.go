package repository

import (
	"context"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestAgentsRepositoryDisconnected(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	repo := NewRepository(nil)
	r.NotNil(repo)

	// ListAgents
	agents, err := repo.ListAgents(ctx)
	r.Error(err)
	r.Nil(agents)

	// ListAgentsByOwner
	agentsByOwner, err := repo.ListAgentsByOwner(ctx, "user-1")
	r.Error(err)
	r.Nil(agentsByOwner)

	// GetAgent
	agent, err := repo.GetAgent(ctx, "agent-1")
	r.Error(err)
	r.Nil(agent)

	// CreateAgent
	err = repo.CreateAgent(ctx, &model.Agent{Name: "Alpha"})
	r.Error(err)

	// UpdateAgent
	err = repo.UpdateAgent(ctx, &model.Agent{Id: "agent-1", Name: "Alpha2"})
	r.Error(err)

	// ActivateAgent
	err = repo.ActivateAgent(ctx, "agent-1", false)
	r.Error(err)
}
