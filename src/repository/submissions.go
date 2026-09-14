package repository

import (
	"context"
	"database/sql"

	"github.com/F4nk1/Agentrix/src/model"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func (r *repository) ListSubmissions(ctx context.Context) ([]model.Submission, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Executing database query to fetch all active submissions")
	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at FROM submissions WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for listing submissions: %s", err)
		return nil, err
	}
	defer rows.Close()

	var submissions []model.Submission
	for rows.Next() {
		var s model.Submission
		if err := rows.Scan(&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed for submissions: %s", err)
			return nil, err
		}
		submissions = append(submissions, s)
	}

	tracer.Debugf(ctx, "Retrieved %d submissions from database", len(submissions))
	return submissions, nil
}

func (r *repository) ListSubmissionsByAgent(ctx context.Context, agentId string) ([]model.Submission, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	tracer.Debugf(ctx, "Querying submissions for agent %s", agentId)
	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at FROM submissions WHERE agent_id = ? AND active = TRUE ORDER BY version DESC`

	rows, err := db.QueryContext(ctx, query, agentId)
	if err != nil {
		tracer.Errorf(ctx, "Database query failed for agent submissions: %s", err)
		return nil, err
	}
	defer rows.Close()

	var submissions []model.Submission
	for rows.Next() {
		var s model.Submission
		if err := rows.Scan(&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt); err != nil {
			tracer.Errorf(ctx, "Database row scan failed: %s", err)
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

	tracer.Debugf(ctx, "Querying database for submission %s", id)
	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at FROM submissions WHERE id = ? AND active = TRUE`

	var s model.Submission
	err = db.QueryRowContext(ctx, query, id).Scan(
		&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			tracer.Warnf(ctx, "Submission %s not found", id)
			return nil, err
		}
		tracer.Errorf(ctx, "Database query failed for submission %s: %s", id, err)
		return nil, err
	}

	tracer.Debugf(ctx, "Successfully retrieved submission %s", id)
	return &s, nil
}

func (r *repository) CreateSubmission(ctx context.Context, submission *model.Submission) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Creating submission '%s' for agent '%s'", submission.Id, submission.AgentId)
	query := `INSERT INTO submissions (id, agent_id, version, code_path, language, status, active, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.ExecContext(ctx, query,
		submission.Id, submission.AgentId, submission.Version, submission.CodePath,
		submission.Language, submission.Status, submission.Active, submission.CreatedAt,
	)
	if err != nil {
		tracer.Errorf(ctx, "Database insert failed for submission %s: %s", submission.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully created submission %s", submission.Id)
	return nil
}

func (r *repository) UpdateSubmission(ctx context.Context, submission *model.Submission) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Updating submission '%s'", submission.Id)
	query := `UPDATE submissions SET status = ?, code_path = ?, active = ? WHERE id = ?`

	_, err = db.ExecContext(ctx, query, submission.Status, submission.CodePath, submission.Active, submission.Id)
	if err != nil {
		tracer.Errorf(ctx, "Database update failed for submission %s: %s", submission.Id, err)
		return err
	}

	tracer.Debugf(ctx, "Successfully updated submission %s", submission.Id)
	return nil
}

func (r *repository) ActivateSubmission(ctx context.Context, id string, isActive bool) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tracer.Debugf(ctx, "Setting submission '%s' active status to %t", id, isActive)
	query := `UPDATE submissions SET active = ? WHERE id = ?`

	_, err = db.ExecContext(ctx, query, isActive, id)
	return err
}
