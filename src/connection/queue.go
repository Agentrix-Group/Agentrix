package connection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/tracer"
	"github.com/google/uuid"
)

var ErrQueueEmpty = errors.New("queue is empty")

type MatchJob struct {
	JobId         string    `json:"job_id"`
	Attempt       int       `json:"attempt"`
	MatchId       string    `json:"match_id"`
	RunId         string    `json:"run_id,omitempty"`
	ContestId     string    `json:"contest_id"`
	GameId        string    `json:"game_id"`
	GameVersion   string    `json:"game_version,omitempty"`
	EngineDigest  string    `json:"engine_digest,omitempty"`
	ConfigHash    string    `json:"config_hash,omitempty"`
	SubmissionIds []string  `json:"submission_ids"`
	Seed          int64     `json:"seed"`
	FencingToken  int64     `json:"fencing_token,omitempty"`
	LeaseUntil    time.Time `json:"lease_until,omitempty"`
}

type JobQueue interface {
	Enqueue(ctx context.Context, job *MatchJob) error
	Dequeue(ctx context.Context) (*MatchJob, error)
	RenewLease(ctx context.Context, job *MatchJob) error
	Complete(ctx context.Context, job *MatchJob) error
	Retry(ctx context.Context, job *MatchJob, cause error) error
	Close() error
	Len() int
}

type inMemoryJobState struct {
	fencingToken int64
	leaseUntil   time.Time
}

type inMemoryJobQueue struct {
	mu           sync.Mutex
	ch           chan *MatchJob
	activeJobs   map[string]*inMemoryJobState
	closed       bool
	tokenCounter int64
}

func NewJobQueue(bufferSize int) JobQueue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &inMemoryJobQueue{
		ch:         make(chan *MatchJob, bufferSize),
		activeJobs: make(map[string]*inMemoryJobState),
	}
}

func (q *inMemoryJobQueue) Enqueue(ctx context.Context, job *MatchJob) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return errors.New("queue is closed")
	}
	if job.RunId == "" {
		job.RunId = uuid.New().String()
	}
	q.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- job:
		return nil
	}
}

func (q *inMemoryJobQueue) Dequeue(ctx context.Context) (*MatchJob, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case job, ok := <-q.ch:
		if !ok {
			return nil, errors.New("queue closed")
		}
		job.Attempt++
		q.mu.Lock()
		q.tokenCounter++
		job.FencingToken = q.tokenCounter
		job.LeaseUntil = time.Now().Add(2 * time.Minute)
		if job.RunId == "" {
			job.RunId = uuid.New().String()
		}
		q.activeJobs[job.JobId] = &inMemoryJobState{
			fencingToken: job.FencingToken,
			leaseUntil:   job.LeaseUntil,
		}
		q.mu.Unlock()
		return job, nil
	}
}

func (q *inMemoryJobQueue) RenewLease(ctx context.Context, job *MatchJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	state, ok := q.activeJobs[job.JobId]
	if !ok || (job.FencingToken > 0 && state.fencingToken != job.FencingToken) {
		return errors.New("match job lease lost or fencing token expired")
	}
	state.leaseUntil = time.Now().Add(2 * time.Minute)
	job.LeaseUntil = state.leaseUntil
	tracer.RecordLeaseRenewal()
	return nil
}

func (q *inMemoryJobQueue) Complete(ctx context.Context, job *MatchJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	state, ok := q.activeJobs[job.JobId]
	if !ok || (job.FencingToken > 0 && state.fencingToken != job.FencingToken) {
		return errors.New("match job reservation was lost before completion (fencing token mismatch)")
	}
	delete(q.activeJobs, job.JobId)
	return nil
}

func (q *inMemoryJobQueue) Retry(ctx context.Context, job *MatchJob, cause error) error {
	q.mu.Lock()
	state, ok := q.activeJobs[job.JobId]
	if ok && job.FencingToken > 0 && state.fencingToken != job.FencingToken {
		q.mu.Unlock()
		return errors.New("match job reservation was lost before retry (fencing token mismatch)")
	}
	delete(q.activeJobs, job.JobId)
	q.mu.Unlock()
	return q.Enqueue(ctx, job)
}

func (q *inMemoryJobQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		q.closed = true
		close(q.ch)
	}
	return nil
}

func (q *inMemoryJobQueue) Len() int { return len(q.ch) }

type postgresJobQueue struct {
	db       *sql.DB
	workerID string
	closed   atomic.Bool
}

func NewPostgresJobQueue(db *sql.DB, workerID string) (JobQueue, error) {
	if db == nil {
		return nil, errors.New("postgres queue requires a database")
	}
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	return &postgresJobQueue{db: db, workerID: workerID}, nil
}

func (q *postgresJobQueue) Enqueue(ctx context.Context, job *MatchJob) error {
	if q.closed.Load() {
		return errors.New("queue is closed")
	}
	if job.RunId == "" {
		job.RunId = uuid.New().String()
	}
	submissions, err := json.Marshal(job.SubmissionIds)
	if err != nil {
		return err
	}
	_, err = q.db.ExecContext(ctx, `
		INSERT INTO match_jobs (
			id, match_id, run_id, contest_id, game_id, submission_ids, seed,
			status, attempt, max_attempts, fencing_token, lease_until, available_at, created_at, updated_at
		) VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6::jsonb, $7, 'pending', 0, 3, 0, NOW() + INTERVAL '2 minutes', NOW(), NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, job.JobId, job.MatchId, job.RunId, job.ContestId, job.GameId, string(submissions), job.Seed)
	return err
}

func (q *postgresJobQueue) Dequeue(ctx context.Context) (*MatchJob, error) {
	if q.closed.Load() {
		return nil, errors.New("queue is closed")
	}

	var job MatchJob
	var submissions []byte
	var runID sql.NullString
	var leaseUntil sql.NullTime
	err := q.db.QueryRowContext(ctx, `
		UPDATE match_jobs
		SET status = 'reserved',
			reserved_by = $1,
			reserved_at = NOW(),
			lease_until = NOW() + INTERVAL '2 minutes',
			fencing_token = fencing_token + 1,
			attempt = attempt + 1,
			updated_at = NOW()
		WHERE id = (
			SELECT id FROM match_jobs
			WHERE attempt < max_attempts
			  AND available_at <= NOW()
			  AND (
				status = 'pending'
				OR (status = 'reserved' AND (lease_until < NOW() OR (lease_until IS NULL AND reserved_at < NOW() - INTERVAL '2 minutes')))
			  )
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, match_id, COALESCE(contest_id, ''), game_id, submission_ids, seed, attempt, run_id, fencing_token, lease_until;
	`, q.workerID).Scan(
		&job.JobId, &job.MatchId, &job.ContestId, &job.GameId,
		&submissions, &job.Seed, &job.Attempt, &runID, &job.FencingToken, &leaseUntil,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrQueueEmpty
	}
	if err != nil {
		return nil, err
	}
	if runID.Valid && runID.String != "" {
		job.RunId = runID.String
	} else {
		job.RunId = uuid.New().String()
	}
	if leaseUntil.Valid {
		job.LeaseUntil = leaseUntil.Time
	}
	if err := json.Unmarshal(submissions, &job.SubmissionIds); err != nil {
		return nil, err
	}
	return &job, nil
}

func (q *postgresJobQueue) RenewLease(ctx context.Context, job *MatchJob) error {
	result, err := q.db.ExecContext(ctx, `
		UPDATE match_jobs
		SET lease_until = NOW() + INTERVAL '2 minutes',
			updated_at = NOW()
		WHERE id = $1 AND status = 'reserved' AND reserved_by = $2 AND (fencing_token = $3 OR $3 = 0)
	`, job.JobId, q.workerID, job.FencingToken)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return errors.New("match job lease lost or fencing token expired")
	}
	job.LeaseUntil = time.Now().Add(2 * time.Minute)
	tracer.RecordLeaseRenewal()
	return nil
}

func (q *postgresJobQueue) Complete(ctx context.Context, job *MatchJob) error {
	result, err := q.db.ExecContext(ctx, `
		UPDATE match_jobs
		SET status = 'completed', reserved_at = NULL, reserved_by = NULL, lease_until = NULL, updated_at = NOW()
		WHERE id = $1 AND status = 'reserved' AND reserved_by = $2 AND (fencing_token = $3 OR $3 = 0)
	`, job.JobId, q.workerID, job.FencingToken)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return errors.New("match job reservation was lost before completion")
	}
	return nil
}

func (q *postgresJobQueue) Retry(ctx context.Context, job *MatchJob, cause error) error {
	backoff := time.Duration(job.Attempt*job.Attempt) * time.Second
	result, err := q.db.ExecContext(ctx, `
		UPDATE match_jobs
		SET status = CASE WHEN attempt >= max_attempts THEN 'failed' ELSE 'pending' END,
			available_at = $2, reserved_at = NULL, reserved_by = NULL, lease_until = NULL,
			last_error = $3, updated_at = NOW()
		WHERE id = $1 AND status = 'reserved' AND reserved_by = $4 AND (fencing_token = $5 OR $5 = 0)
	`, job.JobId, time.Now().UTC().Add(backoff), cause.Error(), q.workerID, job.FencingToken)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return errors.New("match job reservation was lost before retry")
	}
	return nil
}

func (q *postgresJobQueue) Close() error {
	q.closed.Store(true)
	return nil
}

func (q *postgresJobQueue) Len() int {
	var count int
	if err := q.db.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM match_jobs
		WHERE status = 'pending' AND available_at <= NOW()
	`).Scan(&count); err != nil {
		return 0
	}
	return count
}
