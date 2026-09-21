package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

// MatchCommitter handles fenced atomic commits of match results.
type MatchCommitter interface {
	CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error
}

// MatchRunRepository handles execution run lifecycle and audit trails.
type MatchRunRepository interface {
	CreateMatchRun(ctx context.Context, run *model.MatchRun) error
	GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error)
	GetLatestMatchRunByMatch(ctx context.Context, matchId string) (*model.MatchRun, error)
	ListMatchRunsByMatch(ctx context.Context, matchId string) ([]*model.MatchRun, error)
	UpdateMatchRunHeartbeat(ctx context.Context, runID string, heartbeatAt time.Time) error
	UpdateMatchRunStatusCAS(ctx context.Context, runId string, expectedStatus, newStatus model.MatchRunStatus) (bool, error)
	StartMatchRun(ctx context.Context, runId, workerId string, fencingToken int64) error
	FailMatchRun(ctx context.Context, runId string, lastError string) error
}

// IdempotencyRepository handles storage and retrieval of idempotent request results.
type IdempotencyRepository interface {
	GetIdempotencyRecord(ctx context.Context, key string) (*model.IdempotencyRecord, error)
	SaveIdempotencyRecord(ctx context.Context, rec *model.IdempotencyRecord) error
}

func (r *repository) CreateMatchRun(ctx context.Context, run *model.MatchRun) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	query := `
		INSERT INTO match_runs (id, match_id, attempt, worker_id, fencing_token, status, execution_spec, started_at, heartbeat_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	var execSpec any = nil
	if run.ExecutionSpec != "" {
		execSpec = run.ExecutionSpec
	}
	workerID := run.WorkerId
	if workerID == "" {
		workerID = "unassigned"
	}
	attempt := run.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	startedAt := run.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	_, err = db.ExecContext(ctx, query,
		run.Id, run.MatchId, attempt, workerID, run.FencingToken, string(run.Status), execSpec, startedAt, run.HeartbeatAt,
	)
	return err
}

func (r *repository) GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}
	query := `
		SELECT id, match_id, COALESCE(attempt, 1), worker_id, fencing_token, status, COALESCE(execution_spec::text, ''), started_at, finished_at, heartbeat_at, last_error
		FROM match_runs WHERE id = $1
	`
	var run model.MatchRun
	var statusStr string
	var lastError sql.NullString
	err = db.QueryRowContext(ctx, query, id).Scan(
		&run.Id, &run.MatchId, &run.Attempt, &run.WorkerId, &run.FencingToken, &statusStr,
		&run.ExecutionSpec, &run.StartedAt, &run.FinishedAt, &run.HeartbeatAt, &lastError,
	)
	if err != nil {
		return nil, err
	}
	if lastError.Valid {
		run.LastError = lastError.String
	}
	run.Status = model.MatchRunStatus(statusStr)
	return &run, nil
}

func (r *repository) GetLatestMatchRunByMatch(ctx context.Context, matchId string) (*model.MatchRun, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}
	query := `
		SELECT id, match_id, COALESCE(attempt, 1), worker_id, fencing_token, status, COALESCE(execution_spec::text, ''), started_at, finished_at, heartbeat_at, last_error
		FROM match_runs WHERE match_id = $1
		ORDER BY attempt DESC, started_at DESC
		LIMIT 1
	`
	var run model.MatchRun
	var statusStr string
	var lastError sql.NullString
	err = db.QueryRowContext(ctx, query, matchId).Scan(
		&run.Id, &run.MatchId, &run.Attempt, &run.WorkerId, &run.FencingToken, &statusStr,
		&run.ExecutionSpec, &run.StartedAt, &run.FinishedAt, &run.HeartbeatAt, &lastError,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMatchRunNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastError.Valid {
		run.LastError = lastError.String
	}
	run.Status = model.MatchRunStatus(statusStr)
	return &run, nil
}

func (r *repository) ListMatchRunsByMatch(ctx context.Context, matchId string) ([]*model.MatchRun, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}
	query := `
		SELECT id, match_id, COALESCE(attempt, 1), worker_id, fencing_token, status, COALESCE(execution_spec::text, ''), started_at, finished_at, heartbeat_at, last_error
		FROM match_runs WHERE match_id = $1
		ORDER BY attempt ASC, started_at ASC
	`
	rows, err := db.QueryContext(ctx, query, matchId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*model.MatchRun
	for rows.Next() {
		var run model.MatchRun
		var statusStr string
		var lastError sql.NullString
		if err := rows.Scan(
			&run.Id, &run.MatchId, &run.Attempt, &run.WorkerId, &run.FencingToken, &statusStr,
			&run.ExecutionSpec, &run.StartedAt, &run.FinishedAt, &run.HeartbeatAt, &lastError,
		); err != nil {
			return nil, err
		}
		if lastError.Valid {
			run.LastError = lastError.String
		}
		run.Status = model.MatchRunStatus(statusStr)
		runs = append(runs, &run)
	}
	return runs, nil
}

func (r *repository) StartMatchRun(ctx context.Context, runId, workerId string, fencingToken int64) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	query := `
		UPDATE match_runs
		SET status = 'running', worker_id = $2, fencing_token = $3, heartbeat_at = NOW()
		WHERE id = $1 AND status IN ('created', 'dispatching', 'running')
	`
	_, err = db.ExecContext(ctx, query, runId, workerId, fencingToken)
	return err
}

func (r *repository) FailMatchRun(ctx context.Context, runId string, lastError string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	query := `
		UPDATE match_runs
		SET status = 'failed', last_error = $2, finished_at = NOW()
		WHERE id = $1
	`
	_, err = db.ExecContext(ctx, query, runId, lastError)
	return err
}

func (r *repository) GetIdempotencyRecord(ctx context.Context, key string) (*model.IdempotencyRecord, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}
	query := `SELECT key, resource_type, resource_id, response_body::text, status_code, created_at FROM idempotency_keys WHERE key = $1`
	var rec model.IdempotencyRecord
	err = db.QueryRowContext(ctx, query, key).Scan(&rec.Key, &rec.ResourceType, &rec.ResourceId, &rec.ResponseBody, &rec.StatusCode, &rec.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *repository) SaveIdempotencyRecord(ctx context.Context, rec *model.IdempotencyRecord) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	query := `
		INSERT INTO idempotency_keys (key, resource_type, resource_id, response_body, status_code, created_at)
		VALUES ($1, $2, $3, $4::jsonb, $5, NOW())
		ON CONFLICT (key) DO NOTHING
	`
	_, err = db.ExecContext(ctx, query, rec.Key, rec.ResourceType, rec.ResourceId, rec.ResponseBody, rec.StatusCode)
	return err
}

func (r *repository) UpdateMatchRunHeartbeat(ctx context.Context, runID string, heartbeatAt time.Time) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	query := `UPDATE match_runs SET heartbeat_at = $1 WHERE id = $2`
	_, err = db.ExecContext(ctx, query, heartbeatAt, runID)
	return err
}

func (r *repository) UpdateMatchRunStatusCAS(ctx context.Context, runId string, expectedStatus, newStatus model.MatchRunStatus) (bool, error) {
	db, err := r.getDb()
	if err != nil {
		return false, err
	}
	query := `UPDATE match_runs SET status = $1 WHERE id = $2 AND status = $3`
	res, err := db.ExecContext(ctx, query, string(newStatus), runId, string(expectedStatus))
	if err != nil {
		return false, ClassifyDBError(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (r *repository) CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Verify fencing token on match_jobs with row-level lock FOR UPDATE
	var currentFencingToken int64
	var currentJobStatus string
	queryJob := `SELECT fencing_token, status FROM match_jobs WHERE match_id = $1 FOR UPDATE`
	err = tx.QueryRowContext(ctx, queryJob, commit.MatchID).Scan(&currentFencingToken, &currentJobStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrJobNotFound
		}
		return fmt.Errorf("query match job for update: %w", err)
	}

	if currentJobStatus == "completed" {
		return ErrMatchAlreadyCommitted
	}

	if commit.FencingToken > 0 && currentFencingToken != commit.FencingToken {
		return fmt.Errorf("%w: job token is %d, commit token is %d", ErrFencingTokenMismatch, currentFencingToken, commit.FencingToken)
	}

	// 2. Mark match_jobs as completed
	_, err = tx.ExecContext(ctx, `
		UPDATE match_jobs
		SET status = 'completed', reserved_at = NULL, reserved_by = NULL, lease_until = NULL, updated_at = NOW()
		WHERE match_id = $1 AND (fencing_token = $2 OR $2 = 0)
	`, commit.MatchID, commit.FencingToken)
	if err != nil {
		return fmt.Errorf("update match job completed: %w", err)
	}

	runID := commit.RunID
	if runID == "" {
		runID = commit.MatchID + "-run"
	}

	// 3. Record canonical run in match_runs first (so FK from matches, results, replays succeeds)
	finishedAt := commit.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO match_runs (id, match_id, worker_id, fencing_token, status, started_at, finished_at)
		VALUES ($1, $2, $3, $4, 'committed', $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET status = 'committed', finished_at = EXCLUDED.finished_at
	`, runID, commit.MatchID, commit.WorkerID, commit.FencingToken, finishedAt.Add(-time.Second), finishedAt)
	if err != nil {
		return fmt.Errorf("insert canonical match run: %w", err)
	}

	// 4. Update matches table referencing committed_run_id
	_, err = tx.ExecContext(ctx, `
		UPDATE matches
		SET status = $1, replay_id = $2, finished_at = $3, committed_run_id = $4, active = TRUE
		WHERE id = $5
	`, commit.Status, commit.ReplayID, finishedAt, runID, commit.MatchID)
	if err != nil {
		return fmt.Errorf("update match status: %w", err)
	}

	// 5. Insert results associated with run_id and slot_id
	for _, res := range commit.Results {
		var matchRunID any = runID
		if res.MatchRunId != nil && *res.MatchRunId != "" {
			matchRunID = *res.MatchRunId
		}
		var slotID any = res.SlotId
		_, err = tx.ExecContext(ctx, `
			INSERT INTO results (id, match_id, match_run_id, slot_id, submission_id, score, "rank", status, details, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO NOTHING
		`, res.Id, res.MatchId, matchRunID, slotID, res.SubmissionId, res.Score, res.Rank, res.Status, res.Details, res.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert match result for %s: %w", res.SubmissionId, err)
		}
	}

	// 6. Update replays table associated with run_id
	if commit.ReplayID != "" {
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO replays (id, match_id, match_run_id, file_path, duration_ticks, summary, sha256, size_bytes, active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, $9)
			ON CONFLICT (id) DO UPDATE
			SET sha256 = EXCLUDED.sha256, size_bytes = EXCLUDED.size_bytes, file_path = EXCLUDED.file_path, duration_ticks = EXCLUDED.duration_ticks, match_run_id = EXCLUDED.match_run_id
		`, commit.ReplayID, commit.MatchID, runID, commit.ReplayPath, commit.FinalTick, commit.TerminationReason, commit.ReplaySHA256, commit.ReplaySizeBytes, finishedAt)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit match transaction: %w", err)
	}

	return nil
}
