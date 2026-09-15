package service

import (
	"context"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/google/uuid"
)

func (s *service) ListAgents(ctx context.Context) ([]model.Agent, error) {
	return s.repo.ListAgents(ctx)
}

func (s *service) ListAgentsByParticipant(ctx context.Context, participantId string) ([]model.Agent, error) {
	return s.repo.ListAgentsByParticipant(ctx, participantId)
}

func (s *service) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	return s.repo.GetAgent(ctx, id)
}

func (s *service) CreateAgent(ctx context.Context, agent *model.Agent) error {
	if agent.Id == "" {
		agent.Id = uuid.New().String()
	}
	agent.Active = true
	agent.CreatedAt = time.Now().UTC()

	return s.repo.CreateAgent(ctx, agent)
}

func (s *service) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	return s.repo.UpdateAgent(ctx, agent)
}

func (s *service) ActivateAgent(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateAgent(ctx, id, isActive)
}
