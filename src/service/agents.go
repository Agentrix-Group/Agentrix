package service

import (
	"context"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/google/uuid"
)

var (
	ErrGameNotFound       = repository.ErrGameNotFound
	ErrAgentAlreadyExists = repository.ErrAgentAlreadyExists
	ErrSchemaIncompatible = repository.ErrSchemaIncompatible
)

func (s *service) ListAgents(ctx context.Context) ([]model.Agent, error) {
	return s.repo.ListAgents(ctx)
}

func (s *service) ListAgentsByOwner(ctx context.Context, ownerUserId string) ([]model.Agent, error) {
	return s.repo.ListAgentsByOwner(ctx, ownerUserId)
}

func (s *service) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	return s.repo.GetAgent(ctx, id)
}

func (s *service) CreateAgent(ctx context.Context, agent *model.Agent) error {
	if agent.GameId != "starfighter" {
		return ErrUnsupportedGame
	}
	if agent.Id == "" {
		agent.Id = uuid.New().String()
	}
	agent.Active = true
	agent.CreatedAt = time.Now().UTC()

	return s.repo.CreateAgent(ctx, agent)
}

func (s *service) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	if agent.GameId != "starfighter" {
		return ErrUnsupportedGame
	}
	return s.repo.UpdateAgent(ctx, agent)
}

func (s *service) ActivateAgent(ctx context.Context, id string, isActive bool) error {
	return s.repo.ActivateAgent(ctx, id, isActive)
}
