package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

var (
	errMatchNotFound  = model.NotFound("match_not_found", "match not found")
	errRunNotFound    = model.NotFound("run_not_found", "match run not found")
	errReplayNotFound = model.NotFound("replay_not_found", "replay not found")
	// ErrFenced is returned whenever a worker presents a reservation that is
	// no longer the owner of the job (lost lease, stale token, other worker).
	ErrFenced = model.Conflict("reservation_lost", "the job reservation is no longer held by this worker")
)

// ---------------------------------------------------------------------------
// Matches and slots
// ---------------------------------------------------------------------------

const matchColumns = `id, COALESCE(contest_id::text, ''), game_id, mode, state, seed, COALESCE(committed_run_id::text, ''),
	created_by, created_at, updated_at, finished_at`

func scanMatch(row interface{ Scan(...any) error }) (*model.Match, error) {
	var m model.Match
	var finished sql.NullTime
	err := row.Scan(&m.ID, &m.ContestID, &m.GameID, &m.Mode, &m.State, &m.Seed, &m.CommittedRunID,
		&m.CreatedBy, &m.CreatedAt, &m.UpdatedAt, &finished)
	m.FinishedAt = timePtr(finished)
	return &m, err
}

func (q *Queries) CreateMatch(ctx context.Context, m *model.Match) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO matches (id, contest_id, game_id, mode, state, seed, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		m.ID, nullString(m.ContestID), m.GameID, m.Mode, m.State, m.Seed, m.CreatedBy, m.CreatedAt)
	return classify(err)
}

func (q *Queries) CreateSlot(ctx context.Context, mode model.MatchMode, s *model.MatchSlot) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO match_slots (id, match_id, match_mode, slot_index, agent_id, submission_id, contest_entry_id, display_name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		s.ID, s.MatchID, mode, s.SlotIndex, s.AgentID, s.SubmissionID, nullString(s.ContestEntryID), s.DisplayName, s.CreatedAt)
	return classify(err)
}

func (q *Queries) getMatch(ctx context.Context, id, suffix string) (*model.Match, error) {
	if !validUUID(id) {
		return nil, errMatchNotFound
	}
	m, err := scanMatch(q.db.QueryRowContext(ctx, `SELECT `+matchColumns+` FROM matches WHERE id = $1`+suffix, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errMatchNotFound
	}
	if err != nil {
		return nil, classify(err)
	}
	slots, err := q.ListSlots(ctx, id)
	if err != nil {
		return nil, err
	}
	m.Slots = slots
	return m, nil
}

func (q *Queries) GetMatch(ctx context.Context, id string) (*model.Match, error) {
	return q.getMatch(ctx, id, "")
}

func (q *Queries) LockMatch(ctx context.Context, id string) (*model.Match, error) {
	return q.getMatch(ctx, id, " FOR UPDATE")
}

func (q *Queries) ListSlots(ctx context.Context, matchID string) ([]model.MatchSlot, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id, match_id, slot_index, agent_id, submission_id,
		COALESCE(contest_entry_id::text, ''), display_name, created_at
		FROM match_slots WHERE match_id = $1 ORDER BY slot_index`, matchID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.MatchSlot
	for rows.Next() {
		var s model.MatchSlot
		if err := rows.Scan(&s.ID, &s.MatchID, &s.SlotIndex, &s.AgentID, &s.SubmissionID, &s.ContestEntryID,
			&s.DisplayName, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type MatchFilter struct {
	ContestID string
	// PublicOnly hides matches of draft contests.
	PublicOnly bool
	Limit      int
}

func (q *Queries) ListMatches(ctx context.Context, f MatchFilter) ([]model.Match, error) {
	limit := f.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := q.db.QueryContext(ctx, `SELECT m.id, COALESCE(m.contest_id::text, ''), m.game_id, m.mode, m.state, m.seed,
		COALESCE(m.committed_run_id::text, ''), m.created_by, m.created_at, m.updated_at, m.finished_at
		FROM matches m LEFT JOIN contests c ON c.id = m.contest_id
		WHERE ($1 = '' OR m.contest_id::text = $1) AND (NOT $2 OR c.id IS NULL OR c.state <> 'draft')
		ORDER BY m.created_at DESC, m.id LIMIT $3`, f.ContestID, f.PublicOnly, limit)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		slots, err := q.ListSlots(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Slots = slots
	}
	return out, nil
}

// TransitionMatch applies a compare-and-set match state change.
func (q *Queries) TransitionMatch(ctx context.Context, id string, from, to model.MatchState, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE matches SET state = $3, updated_at = $4 WHERE id = $1 AND state = $2`,
		id, from, to, now)
	return expectOne(res, err, model.Conflict("match_state_changed", "match is no longer %s", from))
}

func (q *Queries) FinishMatch(ctx context.Context, id, runID string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE matches SET state = 'finished', committed_run_id = $2, finished_at = $3,
		updated_at = $3 WHERE id = $1 AND state = 'running' AND committed_run_id IS NULL`, id, runID, now)
	return expectOne(res, err, model.Conflict("match_state_changed", "match is no longer running"))
}

// ---------------------------------------------------------------------------
// Runs
// ---------------------------------------------------------------------------

const runColumns = `id, match_id, attempt, state, execution_spec, execution_spec_hash, engine_sha256,
	COALESCE(worker_id, ''), COALESCE(sandbox_runtime, ''), COALESCE(fencing_token, 0),
	created_at, reserved_at, started_at, heartbeat_at, finished_at, COALESCE(error_class, ''), COALESCE(error_message, '')`

func scanRun(row interface{ Scan(...any) error }) (*model.MatchRun, error) {
	var r model.MatchRun
	var spec []byte
	var reserved, started, heartbeat, finished sql.NullTime
	if err := row.Scan(&r.ID, &r.MatchID, &r.Attempt, &r.State, &spec, &r.ExecutionSpecHash, &r.EngineSHA256,
		&r.WorkerID, &r.SandboxRuntime, &r.FencingToken, &r.CreatedAt, &reserved, &started, &heartbeat, &finished,
		&r.ErrorClass, &r.ErrorMessage); err != nil {
		return nil, err
	}
	r.ReservedAt, r.StartedAt, r.HeartbeatAt, r.FinishedAt = timePtr(reserved), timePtr(started), timePtr(heartbeat), timePtr(finished)
	parsed, err := model.ParseExecutionSpec(spec, r.ExecutionSpecHash)
	if err != nil {
		return nil, err
	}
	r.ExecutionSpec = parsed
	return &r, nil
}

// MaxRunAttempt must be called with the match row locked.
func (q *Queries) MaxRunAttempt(ctx context.Context, matchID string) (int, error) {
	var attempt int
	err := q.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt), 0) FROM match_runs WHERE match_id = $1`, matchID).Scan(&attempt)
	return attempt, classify(err)
}

func (q *Queries) CreateRun(ctx context.Context, r *model.MatchRun, specJSON []byte) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO match_runs (id, match_id, attempt, state, execution_spec, execution_spec_hash, engine_sha256, created_at)
		VALUES ($1, $2, $3, 'created', $4, $5, $6, $7)`,
		r.ID, r.MatchID, r.Attempt, specJSON, r.ExecutionSpecHash, r.EngineSHA256, r.CreatedAt)
	return classify(err)
}

func (q *Queries) getRun(ctx context.Context, id, suffix string) (*model.MatchRun, error) {
	if !validUUID(id) {
		return nil, errRunNotFound
	}
	r, err := scanRun(q.db.QueryRowContext(ctx, `SELECT `+runColumns+` FROM match_runs WHERE id = $1`+suffix, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errRunNotFound
	}
	return r, classify(err)
}

func (q *Queries) GetRun(ctx context.Context, id string) (*model.MatchRun, error) {
	return q.getRun(ctx, id, "")
}

func (q *Queries) LockRun(ctx context.Context, id string) (*model.MatchRun, error) {
	return q.getRun(ctx, id, " FOR UPDATE")
}

func (q *Queries) ListRuns(ctx context.Context, matchID string) ([]model.MatchRun, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+runColumns+` FROM match_runs WHERE match_id = $1 ORDER BY attempt`, matchID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.MatchRun
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// SpecJSON returns the persisted spec document of a run (used to clone it for
// an automatic retry without re-deriving any parameter).
func (q *Queries) SpecJSON(ctx context.Context, runID string) ([]byte, error) {
	var raw []byte
	err := q.db.QueryRowContext(ctx, `SELECT execution_spec FROM match_runs WHERE id = $1`, runID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errRunNotFound
	}
	return raw, classify(err)
}

func (q *Queries) ReserveRun(ctx context.Context, runID, workerID, sandbox string, token int64, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE match_runs SET state = 'reserved', worker_id = $2, sandbox_runtime = $3,
		fencing_token = $4, reserved_at = $5, heartbeat_at = $5 WHERE id = $1 AND state = 'created'`,
		runID, workerID, sandbox, token, now)
	return expectOne(res, err, model.Conflict("run_state_changed", "run is no longer created"))
}

func (q *Queries) StartRun(ctx context.Context, runID string, token int64, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE match_runs SET state = 'running', started_at = $3, heartbeat_at = $3
		WHERE id = $1 AND state = 'reserved' AND fencing_token = $2`, runID, token, now)
	return expectOne(res, err, ErrFenced)
}

func (q *Queries) HeartbeatRun(ctx context.Context, runID string, token int64, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE match_runs SET heartbeat_at = $3
		WHERE id = $1 AND fencing_token = $2 AND state IN ('reserved', 'running')`, runID, token, now)
	return expectOne(res, err, ErrFenced)
}

// EndRun moves a live run to a terminal state. token 0 is only valid for
// runs that were never reserved (cancellation of a created run).
func (q *Queries) EndRun(ctx context.Context, runID string, token int64, to model.RunState, class model.ErrorClass, message string, now time.Time) error {
	var res sql.Result
	var err error
	if token == 0 {
		res, err = q.db.ExecContext(ctx, `UPDATE match_runs SET state = $2, error_class = $3, error_message = $4, finished_at = $5
			WHERE id = $1 AND state = 'created'`, runID, to, nullString(string(class)), nullString(truncate(message, 1000)), now)
	} else {
		res, err = q.db.ExecContext(ctx, `UPDATE match_runs SET state = $3, error_class = $4, error_message = $5, finished_at = $6
			WHERE id = $1 AND fencing_token = $2 AND state IN ('reserved', 'running')`,
			runID, token, to, nullString(string(class)), nullString(truncate(message, 1000)), now)
	}
	return expectOne(res, err, ErrFenced)
}

// ---------------------------------------------------------------------------
// Jobs (PostgreSQL queue with leases and fencing tokens)
// ---------------------------------------------------------------------------

const jobColumns = `id, match_id, run_id, engine_sha256, state, available_at, COALESCE(reserved_by, ''), reserved_at,
	lease_until, COALESCE(fencing_token, 0), COALESCE(last_error, ''), created_at, updated_at`

func scanJob(row interface{ Scan(...any) error }) (*model.MatchJob, error) {
	var j model.MatchJob
	var reserved, lease sql.NullTime
	err := row.Scan(&j.ID, &j.MatchID, &j.RunID, &j.EngineSHA256, &j.State, &j.AvailableAt, &j.ReservedBy, &reserved,
		&lease, &j.FencingToken, &j.LastError, &j.CreatedAt, &j.UpdatedAt)
	j.ReservedAt, j.LeaseUntil = timePtr(reserved), timePtr(lease)
	return &j, err
}

func (q *Queries) CreateJob(ctx context.Context, j *model.MatchJob) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO match_jobs (id, match_id, run_id, engine_sha256, state, available_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'pending', $5, $6, $6)`,
		j.ID, j.MatchID, j.RunID, j.EngineSHA256, j.AvailableAt, j.CreatedAt)
	return classify(err)
}

// ActiveJob returns the pending or reserved job of a match, if any.
func (q *Queries) ActiveJob(ctx context.Context, matchID string) (*model.MatchJob, error) {
	j, err := scanJob(q.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM match_jobs
		WHERE match_id = $1 AND state IN ('pending', 'reserved')`, matchID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return j, classify(err)
}

func (q *Queries) ListJobs(ctx context.Context, matchID string) ([]model.MatchJob, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+jobColumns+` FROM match_jobs WHERE match_id = $1 ORDER BY created_at, id`, matchID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.MatchJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

// LockNextPendingJob picks the oldest available job for the engine digest,
// skipping rows locked by other workers.
func (q *Queries) LockNextPendingJob(ctx context.Context, engineSHA string, now time.Time) (*model.MatchJob, error) {
	j, err := scanJob(q.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM match_jobs
		WHERE state = 'pending' AND engine_sha256 = $1 AND available_at <= $2
		ORDER BY available_at, created_at, id LIMIT 1 FOR UPDATE SKIP LOCKED`, engineSHA, now))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return j, classify(err)
}

func (q *Queries) NextFencingToken(ctx context.Context) (int64, error) {
	var token int64
	err := q.db.QueryRowContext(ctx, `SELECT nextval('fencing_token_seq')`).Scan(&token)
	return token, classify(err)
}

func (q *Queries) ReserveJob(ctx context.Context, jobID, workerID string, token int64, leaseUntil, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE match_jobs SET state = 'reserved', reserved_by = $2, reserved_at = $4,
		lease_until = $5, fencing_token = $3, updated_at = $4 WHERE id = $1 AND state = 'pending'`,
		jobID, workerID, token, now, leaseUntil)
	return expectOne(res, err, model.Conflict("job_state_changed", "job is no longer pending"))
}

// RenewLease extends the lease only for the exact owner and token.
func (q *Queries) RenewLease(ctx context.Context, jobID, workerID string, token int64, leaseUntil, now time.Time) error {
	if token <= 0 {
		return ErrFenced
	}
	res, err := q.db.ExecContext(ctx, `UPDATE match_jobs SET lease_until = $4, updated_at = $5
		WHERE id = $1 AND state = 'reserved' AND reserved_by = $2 AND fencing_token = $3 AND lease_until > $5`,
		jobID, workerID, token, leaseUntil, now)
	return expectOne(res, err, ErrFenced)
}

// LockReservedJob locks a job and verifies the complete reservation: state,
// owner, positive exact token, live lease and run identity.
func (q *Queries) LockReservedJob(ctx context.Context, r model.Reservation, now time.Time) (*model.MatchJob, error) {
	if r.FencingToken <= 0 || r.WorkerID == "" || !validUUID(r.JobID) {
		return nil, ErrFenced
	}
	j, err := scanJob(q.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM match_jobs WHERE id = $1 FOR UPDATE`, r.JobID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrFenced
	}
	if err != nil {
		return nil, classify(err)
	}
	if j.State != model.JobReserved || j.ReservedBy != r.WorkerID || j.FencingToken != r.FencingToken ||
		j.RunID != r.RunID || j.MatchID != r.MatchID || j.LeaseUntil == nil || !j.LeaseUntil.After(now) {
		return nil, ErrFenced
	}
	return j, nil
}

func (q *Queries) EndJob(ctx context.Context, jobID string, token int64, to model.JobState, lastError string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE match_jobs SET state = $3, last_error = $4, lease_until = NULL, updated_at = $5
		WHERE id = $1 AND state = 'reserved' AND fencing_token = $2`,
		jobID, token, to, nullString(truncate(lastError, 1000)), now)
	return expectOne(res, err, ErrFenced)
}

func (q *Queries) CancelPendingJob(ctx context.Context, jobID string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE match_jobs SET state = 'cancelled', updated_at = $2
		WHERE id = $1 AND state = 'pending'`, jobID, now)
	return expectOne(res, err, model.Conflict("job_state_changed", "job is no longer pending"))
}

// LockExpiredLeases returns reserved jobs whose lease is over, locked.
func (q *Queries) LockExpiredLeases(ctx context.Context, now time.Time, limit int) ([]model.MatchJob, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+jobColumns+` FROM match_jobs
		WHERE state = 'reserved' AND lease_until <= $1 ORDER BY lease_until, id LIMIT $2 FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.MatchJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Results and replays
// ---------------------------------------------------------------------------

func (q *Queries) InsertResult(ctx context.Context, r *model.Result) error {
	details := r.Details
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO results (id, match_run_id, match_id, slot_id, submission_id, score, rank, outcome, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		r.ID, r.MatchRunID, r.MatchID, r.SlotID, r.SubmissionID, r.Score, r.Rank, r.Outcome, []byte(details), r.CreatedAt)
	return classify(err)
}

func (q *Queries) ListResultsByRun(ctx context.Context, runID string) ([]model.Result, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT r.id, r.match_run_id, r.match_id, r.slot_id, r.submission_id, r.score, r.rank,
		r.outcome, r.details, r.created_at FROM results r JOIN match_slots s ON s.id = r.slot_id
		WHERE r.match_run_id = $1 ORDER BY s.slot_index`, runID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Result
	for rows.Next() {
		var r model.Result
		var details []byte
		if err := rows.Scan(&r.ID, &r.MatchRunID, &r.MatchID, &r.SlotID, &r.SubmissionID, &r.Score, &r.Rank,
			&r.Outcome, &details, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Details = details
		out = append(out, r)
	}
	return out, rows.Err()
}

const replayColumns = `id, match_run_id, match_id, format_version, compression, staging_key, storage_key, sha256, size_bytes,
	frame_count, publication_state, publish_attempts, COALESCE(last_publish_error, ''), next_attempt_at, published_at, created_at`

func scanReplay(row interface{ Scan(...any) error }) (*model.Replay, error) {
	var r model.Replay
	var published sql.NullTime
	err := row.Scan(&r.ID, &r.MatchRunID, &r.MatchID, &r.FormatVersion, &r.Compression, &r.StagingKey, &r.StorageKey,
		&r.SHA256, &r.SizeBytes, &r.FrameCount, &r.Publication, &r.PublishAttempts, &r.LastPublishError,
		&r.NextAttemptAt, &published, &r.CreatedAt)
	r.PublishedAt = timePtr(published)
	return &r, err
}

func (q *Queries) InsertReplay(ctx context.Context, r *model.Replay) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO replays (id, match_run_id, match_id, format_version, compression, staging_key, storage_key, sha256,
			size_bytes, frame_count, publication_state, next_attempt_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'staging', $11, $11)`,
		r.ID, r.MatchRunID, r.MatchID, r.FormatVersion, r.Compression, r.StagingKey, r.StorageKey, r.SHA256,
		r.SizeBytes, r.FrameCount, r.CreatedAt)
	return classify(err)
}

func (q *Queries) GetReplay(ctx context.Context, id string) (*model.Replay, error) {
	if !validUUID(id) {
		return nil, errReplayNotFound
	}
	r, err := scanReplay(q.db.QueryRowContext(ctx, `SELECT `+replayColumns+` FROM replays WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errReplayNotFound
	}
	return r, classify(err)
}

func (q *Queries) GetReplayByRun(ctx context.Context, runID string) (*model.Replay, error) {
	r, err := scanReplay(q.db.QueryRowContext(ctx, `SELECT `+replayColumns+` FROM replays WHERE match_run_id = $1`, runID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return r, classify(err)
}

// LockPendingReplays returns unpublished replays due for a publication
// attempt, locked so that concurrent reconcilers do not publish twice.
func (q *Queries) LockPendingReplays(ctx context.Context, now time.Time, limit int) ([]model.Replay, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+replayColumns+` FROM replays
		WHERE publication_state <> 'published' AND next_attempt_at <= $1
		ORDER BY next_attempt_at, id LIMIT $2 FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Replay
	for rows.Next() {
		r, err := scanReplay(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (q *Queries) MarkReplayPublished(ctx context.Context, id string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE replays SET publication_state = 'published', published_at = $2,
		publish_attempts = publish_attempts + 1, last_publish_error = NULL WHERE id = $1 AND publication_state <> 'published'`, id, now)
	return expectOne(res, err, model.Conflict("replay_already_published", "replay is already published"))
}

func (q *Queries) MarkReplayPublishFailed(ctx context.Context, id, message string, nextAttempt, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE replays SET publication_state = 'publish_failed', last_publish_error = $2,
		publish_attempts = publish_attempts + 1, next_attempt_at = $3 WHERE id = $1 AND publication_state <> 'published'`,
		id, truncate(message, 1000), nextAttempt)
	return expectOne(res, err, model.Conflict("replay_already_published", "replay is already published"))
}

// ReplayCounts summarizes publication states for readiness.
func (q *Queries) ReplayCounts(ctx context.Context) (map[model.ReplayPublication]int, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT publication_state, COUNT(*) FROM replays GROUP BY publication_state`)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	out := map[model.ReplayPublication]int{}
	for rows.Next() {
		var state model.ReplayPublication
		var n int
		if err := rows.Scan(&state, &n); err != nil {
			return nil, err
		}
		out[state] = n
	}
	return out, rows.Err()
}

func (q *Queries) StagingKeysInUse(ctx context.Context) (map[string]bool, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT staging_key FROM replays WHERE publication_state <> 'published'`)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		out[key] = true
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Idempotency
// ---------------------------------------------------------------------------

type IdempotencyRecord struct {
	RequestHash    string
	ResponseStatus int
	ResponseBody   json.RawMessage
	ResourceType   string
	ResourceID     string
}

// FindIdempotency returns the live record for (actor, operation, key). It
// runs inside the command transaction after the key row is locked with
// LockIdempotencyKey, so concurrent requests with the same key serialize.
func (q *Queries) FindIdempotency(ctx context.Context, actorID, operation, key string, now time.Time) (*IdempotencyRecord, error) {
	var rec IdempotencyRecord
	var body []byte
	err := q.db.QueryRowContext(ctx, `SELECT request_hash, response_status, response_body, resource_type, resource_id
		FROM idempotency_keys WHERE actor_user_id = $1 AND operation = $2 AND key = $3 AND expires_at > $4`,
		actorID, operation, key, now).Scan(&rec.RequestHash, &rec.ResponseStatus, &body, &rec.ResourceType, &rec.ResourceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, classify(err)
	}
	rec.ResponseBody = body
	return &rec, nil
}

// LockIdempotencyKey serializes concurrent requests with the same scope and
// key through a transaction-scoped advisory lock, and purges an expired
// record so the key can be reused.
func (q *Queries) LockIdempotencyKey(ctx context.Context, actorID, operation, key string, now time.Time) error {
	if _, err := q.db.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || '|' || $2 || '|' || $3, 0))`,
		actorID, operation, key); err != nil {
		return classify(err)
	}
	_, err := q.db.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE actor_user_id = $1 AND operation = $2 AND key = $3
		AND expires_at <= $4`, actorID, operation, key, now)
	return classify(err)
}

func (q *Queries) SaveIdempotency(ctx context.Context, actorID, operation, key string, rec IdempotencyRecord, now, expires time.Time) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO idempotency_keys (actor_user_id, operation, key, request_hash, response_status, response_body,
			resource_type, resource_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		actorID, operation, key, rec.RequestHash, rec.ResponseStatus, []byte(rec.ResponseBody), rec.ResourceType, rec.ResourceID, now, expires)
	return classify(err)
}

func (q *Queries) PurgeExpiredIdempotency(ctx context.Context, now time.Time) (int64, error) {
	res, err := q.db.ExecContext(ctx, `DELETE FROM idempotency_keys WHERE expires_at <= $1`, now)
	if err != nil {
		return 0, classify(err)
	}
	return res.RowsAffected()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.ValidString(s[:n]) {
		n--
	}
	return s[:n]
}
