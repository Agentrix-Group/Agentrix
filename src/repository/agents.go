package repository

import (
	"context"
	"database/sql"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func (r *repository) ListAgents(ctx context.Context) ([]model.Agent, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, participant_id, game_id, name, description, active, created_at FROM agents WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []model.Agent
	for rows.Next() {
		var a model.Agent
		if err := rows.Scan(&a.Id, &a.ParticipantId, &a.GameId, &a.Name, &a.Description, &a.Active, &a.CreatedAt); err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	return agents, nil
}

func (r *repository) ListAgentsByParticipant(ctx context.Context, participantId string) ([]model.Agent, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, participant_id, game_id, name, description, active, created_at FROM agents WHERE participant_id = $1 AND active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query, participantId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []model.Agent
	for rows.Next() {
		var a model.Agent
		if err := rows.Scan(&a.Id, &a.ParticipantId, &a.GameId, &a.Name, &a.Description, &a.Active, &a.CreatedAt); err != nil {
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

	query := `SELECT id, participant_id, game_id, name, description, active, created_at FROM agents WHERE id = $1 AND active = TRUE`

	var a model.Agent
	err = db.QueryRowContext(ctx, query, id).Scan(
		&a.Id, &a.ParticipantId, &a.GameId, &a.Name, &a.Description, &a.Active, &a.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &a, nil
}

func (r *repository) CreateAgent(ctx context.Context, agent *model.Agent) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `INSERT INTO agents (id, participant_id, game_id, name, description, active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = db.ExecContext(ctx, query,
		agent.Id, agent.ParticipantId, agent.GameId, agent.Name,
		agent.Description, agent.Active, agent.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) UpdateAgent(ctx context.Context, agent *model.Agent) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE agents SET name = $1, description = $2, active = $3 WHERE id = $4`

	_, err = db.ExecContext(ctx, query, agent.Name, agent.Description, agent.Active, agent.Id)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) ActivateAgent(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE agents SET active = $1 WHERE id = $2`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
