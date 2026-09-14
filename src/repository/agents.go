package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func (r *repository) ListAgents(ctx context.Context) ([]model.Agent, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch all active agents")
	query := `SELECT id, participant_id, game_id, name, description, active, created_at FROM agents WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing agents: %s", err)
		return nil, err
	}
	defer rows.Close()

	var agents []model.Agent
	for rows.Next() {
		var a model.Agent
		if err := rows.Scan(&a.Id, &a.ParticipantId, &a.GameId, &a.Name, &a.Description, &a.Active, &a.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for agents: %s", err)
			return nil, err
		}
		agents = append(agents, a)
	}

	tracer.Debugf(ctx, "Retrieved %d agents from database", len(agents))
	return agents, nil
}

func (r *repository) ListAgentsByParticipant(ctx context.Context, participantId string) ([]model.Agent, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying agents for participant %s", participantId)
	query := `SELECT id, participant_id, game_id, name, description, active, created_at FROM agents WHERE participant_id = $1 AND active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query, participantId)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for participant agents: %s", err)
		return nil, err
	}
	defer rows.Close()

	var agents []model.Agent
	for rows.Next() {
		var a model.Agent
		if err := rows.Scan(&a.Id, &a.ParticipantId, &a.GameId, &a.Name, &a.Description, &a.Active, &a.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed: %s", err)
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

func (r *repository) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying database for agent %s", id)
	query := `SELECT id, participant_id, game_id, name, description, active, created_at FROM agents WHERE id = $1 AND active = TRUE`

	var a model.Agent
	err = db.QueryRowContext(ctx, query, id).Scan(
		&a.Id, &a.ParticipantId, &a.GameId, &a.Name, &a.Description, &a.Active, &a.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Agent %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for agent %s: %s", id, err)
		return nil, err
	}

	tracer.Debugf(ctx, "Successfully retrieved agent %s", id)
	return &a, nil
}

func (r *repository) CreateAgent(ctx context.Context, agent *model.Agent) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Creating agent '%s'", agent.Name)
	query := `INSERT INTO agents (id, participant_id, game_id, name, description, active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.ExecContext(ctx, query,
		agent.Id, agent.ParticipantId, agent.GameId, agent.Name,
		agent.Description, agent.Active, agent.CreatedAt,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database insert failed for agent %s: %s", agent.Name, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully created agent %s", agent.Id)
	return nil
}

func (r *repository) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Updating agent '%s'", agent.Id)
	query := `UPDATE agents SET name = $1, description = $2, active = $3 WHERE id = $4`

	_, err = db.ExecContext(ctx, query, agent.Name, agent.Description, agent.Active, agent.Id)
	if err != nil {
		tracer.Errorf(ctx, "Database update failed for agent %s: %s", agent.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully updated agent %s", agent.Id)
	return nil
}

func (r *repository) ActivateAgent(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Setting agent '%s' active status to %t", id, isActive)
	query := `UPDATE agents SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
