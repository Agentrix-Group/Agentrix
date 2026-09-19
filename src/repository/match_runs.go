package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/F4nk1/Agentrix/src/model"
)

// MatchCommitter handles fenced atomic commits of match results.
type MatchCommitter interface {
	CommitMatchResult(ctx context.Context, commit model.MatchResultCommit) error
}

// MatchRunRepository handles execution run lifecycle and audit trails.
type MatchRunRepository interface {
	CreateMatchRun(ctx context.Context, run *model.MatchRun) error
	GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error)
	UpdateMatchRunHeartbeat(ctx context.Context, runID string, heartbeatAt time.Time) error
}

func (r *repository) CreateMatchRun(ctx context.Context, run *model.MatchRun) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}
	query := `
		INSERT INTO match_runs (id, match_id, worker_id, fencing_token, status, started_at, heartbeat_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = db.ExecContext(ctx, query,
		run.Id, run.MatchId, run.WorkerId, run.FencingToken, string(run.Status), run.StartedAt, run.HeartbeatAt,
	)
	return err
}

func (r *repository) GetMatchRun(ctx context.Context, id string) (*model.MatchRun, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}
	query := `
		SELECT id, match_id, worker_id, fencing_token, status, started_at, finished_at, heartbeat_at, last_error
		FROM match_runs WHERE id = $1
	`
	var run model.MatchRun
	var statusStr string
	err = db.QueryRowContext(ctx, query, id).Scan(
		&run.Id, &run.MatchId, &run.WorkerId, &run.FencingToken, &statusStr,
		&run.StartedAt, &run.FinishedAt, &run.HeartbeatAt, &run.LastError,
	)
	if err != nil {
		return nil, err
	}
	run.Status = model.MatchRunStatus(statusStr)
	return &run, nil
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

	// 3. Update matches table
	finishedAt := commit.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE matches
		SET status = $1, replay_id = $2, finished_at = $3, active = TRUE
		WHERE id = $4
	`, commit.Status, commit.ReplayID, finishedAt, commit.MatchID)
	if err != nil {
		return fmt.Errorf("update match status: %w", err)
	}

	// 4. Record canonical run in match_runs
	runID := commit.RunID
	if runID == "" {
		runID = commit.MatchID + "-run"
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

	// 5. Insert results
	for _, res := range commit.Results {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO results (id, match_id, submission_id, score, "rank", status, details, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO NOTHING
		`, res.Id, res.MatchId, res.SubmissionId, res.Score, res.Rank, res.Status, res.Details, res.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert match result for %s: %w", res.SubmissionId, err)
		}
	}

	// 6. Update replays table if replay record exists
	if commit.ReplayID != "" {
		_, _ = tx.ExecContext(ctx, `
			INSERT INTO replays (id, match_id, file_path, duration_ticks, summary, sha256, size_bytes, active, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8)
			ON CONFLICT (id) DO UPDATE
			SET sha256 = EXCLUDED.sha256, size_bytes = EXCLUDED.size_bytes, file_path = EXCLUDED.file_path, duration_ticks = EXCLUDED.duration_ticks
		`, commit.ReplayID, commit.MatchID, commit.ReplayPath, commit.FinalTick, commit.TerminationReason, commit.ReplaySHA256, commit.ReplaySizeBytes, finishedAt)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit match transaction: %w", err)
	}

	return nil
}
