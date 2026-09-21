package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/google/uuid"
)

var (
	errGameNotFound       = model.NotFound("game_not_found", "game not found")
	errAgentNotFound      = model.NotFound("agent_not_found", "agent not found")
	errSubmissionNotFound = model.NotFound("submission_not_found", "submission not found")
)

func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// UpsertGame registers a loaded game module. Only availability metadata is
// stored; rules stay in the module.
func (q *Queries) UpsertGame(ctx context.Context, g model.Game, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO games (id, name, version, min_players, max_players, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'active', $6, $6)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, version = EXCLUDED.version,
			min_players = EXCLUDED.min_players, max_players = EXCLUDED.max_players, updated_at = EXCLUDED.updated_at`,
		g.ID, g.Name, g.Version, g.MinPlayers, g.MaxPlayers, now)
	return classify(err)
}

const gameColumns = `id, name, version, min_players, max_players, status, created_at, updated_at`

func scanGame(row interface{ Scan(...any) error }) (*model.Game, error) {
	var g model.Game
	err := row.Scan(&g.ID, &g.Name, &g.Version, &g.MinPlayers, &g.MaxPlayers, &g.Status, &g.CreatedAt, &g.UpdatedAt)
	return &g, err
}

func (q *Queries) GetGame(ctx context.Context, id string) (*model.Game, error) {
	g, err := scanGame(q.db.QueryRowContext(ctx, `SELECT `+gameColumns+` FROM games WHERE id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errGameNotFound
	}
	return g, classify(err)
}

func (q *Queries) ListGames(ctx context.Context) ([]model.Game, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+gameColumns+` FROM games ORDER BY id`)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Game
	for rows.Next() {
		g, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

// RegisterWorker records the engine artifact a worker runs and its heartbeat.
func (q *Queries) RegisterWorker(ctx context.Context, w model.WorkerInfo, artifact model.EngineArtifact, now time.Time) error {
	if _, err := q.db.ExecContext(ctx, `
		INSERT INTO engine_artifacts (sha256, game_id, game_version, engine_version, protocol_version, source_commit, first_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (sha256) DO NOTHING`,
		artifact.SHA256, artifact.GameID, artifact.GameVersion, artifact.EngineVersion, artifact.ProtocolVersion,
		artifact.SourceCommit, now); err != nil {
		return classify(err)
	}
	var stored model.EngineArtifact
	if err := q.db.QueryRowContext(ctx, `SELECT game_id, game_version, engine_version, protocol_version
		FROM engine_artifacts WHERE sha256 = $1`, artifact.SHA256).Scan(
		&stored.GameID, &stored.GameVersion, &stored.EngineVersion, &stored.ProtocolVersion); err != nil {
		return classify(err)
	}
	if stored.GameID != artifact.GameID || stored.GameVersion != artifact.GameVersion ||
		stored.EngineVersion != artifact.EngineVersion || stored.ProtocolVersion != artifact.ProtocolVersion {
		return model.Conflict("engine_artifact_mismatch", "engine digest %s is already registered with different metadata", artifact.SHA256)
	}
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO workers (id, engine_sha256, sandbox_runtime, started_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (id) DO UPDATE SET engine_sha256 = EXCLUDED.engine_sha256, sandbox_runtime = EXCLUDED.sandbox_runtime,
			started_at = EXCLUDED.started_at, last_seen_at = EXCLUDED.last_seen_at`,
		w.ID, artifact.SHA256, w.SandboxRuntime, now)
	return classify(err)
}

func (q *Queries) TouchWorker(ctx context.Context, workerID string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE workers SET last_seen_at = $2 WHERE id = $1`, workerID, now)
	return expectOne(res, err, model.NotFound("worker_not_found", "worker is not registered"))
}

// SelectEngineArtifact picks the engine for a new run: the artifact of the
// most recently seen live worker for the game version and protocol.
func (q *Queries) SelectEngineArtifact(ctx context.Context, gameID, gameVersion, protocol string, liveSince time.Time) (*model.EngineArtifact, error) {
	var a model.EngineArtifact
	err := q.db.QueryRowContext(ctx, `
		SELECT e.sha256, e.game_id, e.game_version, e.engine_version, e.protocol_version, e.source_commit
		FROM engine_artifacts e JOIN workers w ON w.engine_sha256 = e.sha256
		WHERE e.game_id = $1 AND e.game_version = $2 AND e.protocol_version = $3 AND w.last_seen_at >= $4
		ORDER BY w.last_seen_at DESC, e.sha256 LIMIT 1`, gameID, gameVersion, protocol, liveSince).Scan(
		&a.SHA256, &a.GameID, &a.GameVersion, &a.EngineVersion, &a.ProtocolVersion, &a.SourceCommit)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.Unavailable("engine_unavailable",
			"no live worker offers an engine for %s %s (%s)", gameID, gameVersion, protocol)
	}
	return &a, classify(err)
}

func (q *Queries) ListWorkers(ctx context.Context) ([]model.WorkerInfo, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT w.id, w.engine_sha256, e.engine_version, w.sandbox_runtime, w.started_at, w.last_seen_at
		FROM workers w JOIN engine_artifacts e ON e.sha256 = w.engine_sha256 ORDER BY w.last_seen_at DESC, w.id`)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.WorkerInfo
	for rows.Next() {
		var w model.WorkerInfo
		if err := rows.Scan(&w.ID, &w.EngineSHA256, &w.EngineVersion, &w.SandboxRuntime, &w.StartedAt, &w.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------

const agentColumns = `a.id, a.owner_user_id, u.username, a.game_id, a.name, a.description, a.status, a.created_at, a.updated_at`

func scanAgent(row interface{ Scan(...any) error }) (*model.Agent, error) {
	var a model.Agent
	err := row.Scan(&a.ID, &a.OwnerUserID, &a.OwnerName, &a.GameID, &a.Name, &a.Description, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	return &a, err
}

func (q *Queries) CreateAgent(ctx context.Context, a *model.Agent) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO agents (id, owner_user_id, game_id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		a.ID, a.OwnerUserID, a.GameID, a.Name, a.Description, a.Status, a.CreatedAt)
	return classify(err)
}

func (q *Queries) getAgent(ctx context.Context, id, suffix string) (*model.Agent, error) {
	if !validUUID(id) {
		return nil, errAgentNotFound
	}
	a, err := scanAgent(q.db.QueryRowContext(ctx, `SELECT `+agentColumns+`
		FROM agents a JOIN users u ON u.id = a.owner_user_id WHERE a.id = $1`+suffix, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errAgentNotFound
	}
	return a, classify(err)
}

func (q *Queries) GetAgent(ctx context.Context, id string) (*model.Agent, error) {
	return q.getAgent(ctx, id, "")
}

func (q *Queries) LockAgent(ctx context.Context, id string) (*model.Agent, error) {
	return q.getAgent(ctx, id, " FOR UPDATE OF a")
}

// ListAgents lists agents of one owner, or of every owner when ownerID is empty.
func (q *Queries) ListAgents(ctx context.Context, ownerID string) ([]model.Agent, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+agentColumns+`
		FROM agents a JOIN users u ON u.id = a.owner_user_id
		WHERE ($1 = '' OR a.owner_user_id::text = $1) ORDER BY a.created_at, a.id`, ownerID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (q *Queries) UpdateAgentDetails(ctx context.Context, id, name, description string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE agents SET name = $2, description = $3, updated_at = $4 WHERE id = $1`,
		id, name, description, now)
	return expectOne(res, err, errAgentNotFound)
}

func (q *Queries) UpdateAgentStatus(ctx context.Context, id string, status model.AgentStatus, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE agents SET status = $2, updated_at = $3 WHERE id = $1`, id, status, now)
	return expectOne(res, err, errAgentNotFound)
}

// ---------------------------------------------------------------------------
// Submissions
// ---------------------------------------------------------------------------

const submissionColumns = `id, agent_id, version, status, runtime, entrypoint, artifact_key, artifact_sha256, size_bytes,
	manifest, COALESCE(admission_error, ''), created_by, created_at, updated_at`

func scanSubmission(row interface{ Scan(...any) error }) (*model.Submission, error) {
	var s model.Submission
	var manifest []byte
	err := row.Scan(&s.ID, &s.AgentID, &s.Version, &s.Status, &s.Runtime, &s.Entrypoint, &s.ArtifactKey, &s.ArtifactSHA256,
		&s.SizeBytes, &manifest, &s.AdmissionError, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	s.Manifest = manifest
	return &s, err
}

// NextSubmissionVersion must run in a transaction that locked the agent row;
// the lock serializes concurrent uploads and UNIQUE(agent_id, version) backs it.
func (q *Queries) NextSubmissionVersion(ctx context.Context, agentID string) (int, error) {
	var version int
	err := q.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM submissions WHERE agent_id = $1`, agentID).Scan(&version)
	return version, classify(err)
}

func (q *Queries) CreateSubmission(ctx context.Context, s *model.Submission) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO submissions (id, agent_id, version, status, runtime, entrypoint, artifact_key, artifact_sha256,
			size_bytes, manifest, admission_error, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)`,
		s.ID, s.AgentID, s.Version, s.Status, s.Runtime, s.Entrypoint, s.ArtifactKey, s.ArtifactSHA256,
		s.SizeBytes, []byte(s.Manifest), nullString(s.AdmissionError), s.CreatedBy, s.CreatedAt)
	return classify(err)
}

func (q *Queries) getSubmission(ctx context.Context, id, suffix string) (*model.Submission, error) {
	if !validUUID(id) {
		return nil, errSubmissionNotFound
	}
	s, err := scanSubmission(q.db.QueryRowContext(ctx, `SELECT `+submissionColumns+` FROM submissions WHERE id = $1`+suffix, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errSubmissionNotFound
	}
	return s, classify(err)
}

func (q *Queries) GetSubmission(ctx context.Context, id string) (*model.Submission, error) {
	return q.getSubmission(ctx, id, "")
}

func (q *Queries) LockSubmission(ctx context.Context, id string) (*model.Submission, error) {
	return q.getSubmission(ctx, id, " FOR SHARE")
}

func (q *Queries) ListSubmissions(ctx context.Context, agentID string) ([]model.Submission, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+submissionColumns+` FROM submissions WHERE agent_id = $1 ORDER BY version`, agentID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Submission
	for rows.Next() {
		s, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

// FinishAdmission moves a validating submission to ready or rejected.
func (q *Queries) FinishAdmission(ctx context.Context, id string, status model.SubmissionStatus, admissionError string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE submissions SET status = $2, admission_error = $3, updated_at = $4
		WHERE id = $1 AND status = 'validating'`, id, status, nullString(admissionError), now)
	return expectOne(res, err, model.Conflict("submission_not_validating", "submission is not awaiting admission"))
}

func (q *Queries) DisableSubmission(ctx context.Context, id string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE submissions SET status = 'disabled', updated_at = $2
		WHERE id = $1 AND status = 'ready'`, id, now)
	return expectOne(res, err, model.InvalidTransition("submission", "not ready", model.SubmissionDisabled))
}

// RejectStaleAdmissions closes admissions interrupted by a crash.
func (q *Queries) RejectStaleAdmissions(ctx context.Context, olderThan, now time.Time) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE submissions SET status = 'rejected',
		admission_error = 'admission was interrupted before completion; upload the bundle again', updated_at = $2
		WHERE status = 'validating' AND created_at < $1`, olderThan, now)
	if err != nil {
		return 0, classify(err)
	}
	return res.RowsAffected()
}

// ReferencedArtifactKeys lists every artifact key referenced by a submission.
func (q *Queries) ReferencedArtifactKeys(ctx context.Context) (map[string]bool, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT DISTINCT artifact_key FROM submissions`)
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
