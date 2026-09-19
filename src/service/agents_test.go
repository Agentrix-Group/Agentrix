package service

import (
	"context"
	"testing"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/repository"
	"github.com/stretchr/testify/require"
)

type mockAgentsRepo struct {
	repository.Repository
	listAgentsFn              func(ctx context.Context) ([]model.Agent, error)
	listAgentsByParticipantFn func(ctx context.Context, participantId string) ([]model.Agent, error)
	getAgentFn                func(ctx context.Context, id string) (*model.Agent, error)
	createAgentFn             func(ctx context.Context, agent *model.Agent) error
	updateAgentFn             func(ctx context.Context, agent *model.Agent) error
	activateAgentFn           func(ctx context.Context, id string, isActive bool) error
}

func (m *mockAgentsRepo) ListAgents(ctx context.Context) ([]model.Agent, error) {
	if m.listAgentsFn != nil {
		return m.listAgentsFn(ctx)
	}
	return nil, nil
}

func (m *mockAgentsRepo) ListAgentsByParticipant(ctx context.Context, participantId string) ([]model.Agent, error) {
	if m.listAgentsByParticipantFn != nil {
		return m.listAgentsByParticipantFn(ctx, participantId)
	}
	return nil, nil
}

func (m *mockAgentsRepo) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	if m.getAgentFn != nil {
		return m.getAgentFn(ctx, id)
	}
	return &model.Agent{Id: id, Name: "TestAgent"}, nil
}

func (m *mockAgentsRepo) CreateAgent(ctx context.Context, agent *model.Agent) error {
	if m.createAgentFn != nil {
		return m.createAgentFn(ctx, agent)
	}
	return nil
}

func (m *mockAgentsRepo) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	if m.updateAgentFn != nil {
		return m.updateAgentFn(ctx, agent)
	}
	return nil
}

func (m *mockAgentsRepo) ActivateAgent(ctx context.Context, id string, isActive bool) error {
	if m.activateAgentFn != nil {
		return m.activateAgentFn(ctx, id, isActive)
	}
	return nil
}

func TestAgentsService(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	createdAgents := make([]*model.Agent, 0)
	mockRepo := &mockAgentsRepo{
		listAgentsFn: func(ctx context.Context) ([]model.Agent, error) {
			return []model.Agent{{Id: "a1", Name: "Bot1"}}, nil
		},
		listAgentsByParticipantFn: func(ctx context.Context, participantId string) ([]model.Agent, error) {
			return []model.Agent{{Id: "a1", ParticipantId: participantId}}, nil
		},
		createAgentFn: func(ctx context.Context, agent *model.Agent) error {
			createdAgents = append(createdAgents, agent)
			return nil
		},
	}

	svc := NewService(mockRepo, nil, nil)

	// ListAgents
	agents, err := svc.ListAgents(ctx)
	r.NoError(err)
	r.Len(agents, 1)

	// ListAgentsByParticipant
	partAgents, err := svc.ListAgentsByParticipant(ctx, "part-1")
	r.NoError(err)
	r.Len(partAgents, 1)

	// GetAgent
	agent, err := svc.GetAgent(ctx, "a1")
	r.NoError(err)
	r.Equal("a1", agent.Id)

	// CreateAgent
	newAgent := &model.Agent{Name: "NewBot", ParticipantId: "part-1", GameId: "starfighter"}
	err = svc.CreateAgent(ctx, newAgent)
	r.NoError(err)
	r.NotEmpty(newAgent.Id)
	r.True(newAgent.Active)
	r.Len(createdAgents, 1)

	// UpdateAgent & ActivateAgent
	r.NoError(svc.UpdateAgent(ctx, newAgent))
	r.NoError(svc.ActivateAgent(ctx, newAgent.Id, false))
	r.ErrorIs(svc.CreateAgent(ctx, &model.Agent{Name: "Other", GameId: "other-game"}), ErrUnsupportedGame)
}
