package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

func (r *repository) ListSubmissions(ctx context.Context) ([]model.Submission, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at, COALESCE(error_detail, '') FROM submissions WHERE active = TRUE ORDER BY created_at DESC`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []model.Submission
	for rows.Next() {
		var s model.Submission
		if err := rows.Scan(&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt, &s.ErrorDetail); err != nil {
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

	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at, COALESCE(error_detail, '') FROM submissions WHERE agent_id = $1 AND active = TRUE ORDER BY version DESC`

	rows, err := db.QueryContext(ctx, query, agentId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []model.Submission
	for rows.Next() {
		var s model.Submission
		if err := rows.Scan(&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt, &s.ErrorDetail); err != nil {
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

	query := `SELECT id, agent_id, version, code_path, language, status, active, created_at, COALESCE(error_detail, '') FROM submissions WHERE id = $1 AND active = TRUE`

	var s model.Submission
	err = db.QueryRowContext(ctx, query, id).Scan(
		&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt, &s.ErrorDetail,
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

	query := `INSERT INTO submissions (id, agent_id, version, code_path, language, status, active, created_at, error_detail) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF($9, ''))`

	_, err = db.ExecContext(ctx, query,
		submission.Id, submission.AgentId, submission.Version, submission.CodePath,
		submission.Language, submission.Status, submission.Active, submission.CreatedAt,
		submission.ErrorDetail,
	)
	if err != nil {
		return err
	}

	return nil
}

// SubmissionAdmissionRepository es la cola de admisión de bots en el worker
// (ADR-0014, N4): las submissions en 'validating' esperan su prueba.
type SubmissionAdmissionRepository interface {
	ClaimNextAdmission(ctx context.Context, staleAfter time.Duration) (*model.Submission, error)
	FinishAdmission(ctx context.Context, id, status, detail string) error
}

// ClaimNextAdmission reserva la submission 'validating' más antigua para un
// worker. SKIP LOCKED impide que dos workers prueben el mismo bot; una
// reserva más vieja que staleAfter (worker caído) se puede retomar.
// Devuelve nil si no hay trabajo.
func (r *repository) ClaimNextAdmission(ctx context.Context, staleAfter time.Duration) (*model.Submission, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}
	query := `UPDATE submissions SET admission_claimed_at = NOW()
		WHERE id = (
			SELECT id FROM submissions
			WHERE status = 'validating' AND active = TRUE
			  AND (admission_claimed_at IS NULL OR admission_claimed_at < NOW() - make_interval(secs => $1))
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, agent_id, version, code_path, language, status, active, created_at`
	var s model.Submission
	err = db.QueryRowContext(ctx, query, staleAfter.Seconds()).Scan(
		&s.Id, &s.AgentId, &s.Version, &s.CodePath, &s.Language, &s.Status, &s.Active, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FinishAdmission guarda el resultado de la prueba. Solo cambia una
// submission que sigue en 'validating'.
func (r *repository) FinishAdmission(ctx context.Context, id, status, detail string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx,
		`UPDATE submissions SET status = $1, error_detail = NULLIF($2, ''), admission_claimed_at = NULL WHERE id = $3 AND status = 'validating'`,
		status, detail, id)
	return err
}
