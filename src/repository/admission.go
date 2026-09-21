package repository

import (
	"context"
	"database/sql"
	"errors"
)

// LockNextValidatingSubmission picks one submission awaiting admission and
// locks it for the rest of the transaction (other workers skip it).
func (q *Queries) LockNextValidatingSubmission(ctx context.Context) (string, error) {
	var id string
	err := q.db.QueryRowContext(ctx, `SELECT id FROM submissions WHERE status = 'validating'
		ORDER BY created_at, id LIMIT 1 FOR UPDATE SKIP LOCKED`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, classify(err)
}

// QueueStats summarizes the job queue for readiness reporting.
type QueueStats struct {
	Pending         int
	Reserved        int
	ExpiredLeases   int
	OldestPendingMs int64
}

func (q *Queries) QueueStats(ctx context.Context) (QueueStats, error) {
	var s QueueStats
	err := q.db.QueryRowContext(ctx, `SELECT
		COUNT(*) FILTER (WHERE state = 'pending'),
		COUNT(*) FILTER (WHERE state = 'reserved'),
		COUNT(*) FILTER (WHERE state = 'reserved' AND lease_until <= NOW()),
		COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(available_at) FILTER (WHERE state = 'pending'))) * 1000, 0)::bigint
		FROM match_jobs`).Scan(&s.Pending, &s.Reserved, &s.ExpiredLeases, &s.OldestPendingMs)
	return s, classify(err)
}

// OrphanCheck reports rows that contradict each other; the smoke test and
// readiness require every count to be zero.
type OrphanCheck struct {
	ActiveJobsWithoutLiveRun int
	RunningMatchesWithoutJob int
	QueuedMatchesWithoutJob  int
	CommittedWithoutResults  int
}

func (q *Queries) OrphanCheck(ctx context.Context) (OrphanCheck, error) {
	var o OrphanCheck
	err := q.db.QueryRowContext(ctx, `SELECT
		(SELECT COUNT(*) FROM match_jobs j JOIN match_runs r ON r.id = j.run_id
		   WHERE j.state IN ('pending', 'reserved') AND r.state NOT IN ('created', 'reserved', 'running')),
		(SELECT COUNT(*) FROM matches m WHERE m.state = 'running'
		   AND NOT EXISTS (SELECT 1 FROM match_jobs j WHERE j.match_id = m.id AND j.state = 'reserved')),
		(SELECT COUNT(*) FROM matches m WHERE m.state = 'queued'
		   AND NOT EXISTS (SELECT 1 FROM match_jobs j WHERE j.match_id = m.id AND j.state IN ('pending', 'reserved'))),
		(SELECT COUNT(*) FROM matches m WHERE m.state = 'finished'
		   AND (SELECT COUNT(*) FROM results r WHERE r.match_run_id = m.committed_run_id)
		     <> (SELECT COUNT(*) FROM match_slots s WHERE s.match_id = m.id))`).Scan(
		&o.ActiveJobsWithoutLiveRun, &o.RunningMatchesWithoutJob, &o.QueuedMatchesWithoutJob, &o.CommittedWithoutResults)
	return o, classify(err)
}
