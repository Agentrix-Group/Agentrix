package repository

import (
	"context"
	"database/sql"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func (r *repository) ListSubmissions(ctx context.Context) ([]model.Submission, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at FROM submissions WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []model.Submission
	for rows.Next() {
		var s model.Submission
		if err := rows.Scan(&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt); err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	return submissions, nil
}

func (r *repository) ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at FROM submissions WHERE agent_id = $1 AND active = TRUE ORDER BY version DESC`

	rows, err := db.QueryContext(ctx, query, agentId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []model.Submission
	for rows.Next() {
		var s model.Submission
		if err := rows.Scan(&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt); err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}
	return submissions, nil
}

func (r *repository) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at FROM submissions WHERE id = $1 AND active = TRUE`

	var s model.Submission
	err = db.QueryRowContext(ctx, query, id).Scan(
		&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, err
		}
		return nil, err
	}

	return &s, nil
}

func (r *repository) CreateSubmission(ctx context.Context, submission *model.Submission) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `INSERT INTO submissions (id, agent_id, version, code_path, language, status, active, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = db.ExecContext(ctx, query,
		submission.Id, submission.AgentId, submission.Version, submission.CodePath,
		submission.Language, submission.Status, submission.Active, submission.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}
