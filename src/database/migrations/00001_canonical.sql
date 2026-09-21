-- Agentrix canonical baseline.
--
-- Single source of truth for the chain
-- User -> Agent -> Submission -> ContestEntry -> Match -> MatchSlot -> MatchRun
--      -> Result/Replay -> RankingSnapshot.
--
-- Role and capability rows are policy reference data (the RBAC matrix), not
-- demo data. Demo users, contests and bots are created by cmd/bootstrap
-- through the use cases, never by a schema migration.

-- +goose Up
-- +goose StatementBegin

-- ---------------------------------------------------------------------------
-- Shared trigger functions
-- ---------------------------------------------------------------------------

CREATE FUNCTION agentrix_forbid_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'table % is append-only (% forbidden)', TG_TABLE_NAME, TG_OP
        USING ERRCODE = 'check_violation';
END;
$$;

-- ---------------------------------------------------------------------------
-- Identity and RBAC
-- ---------------------------------------------------------------------------

CREATE TABLE roles (
    id          TEXT PRIMARY KEY CHECK (id ~ '^[a-z][a-z_]{1,31}$'),
    description TEXT NOT NULL
);

CREATE TABLE capabilities (
    id          TEXT PRIMARY KEY CHECK (id ~ '^[a-z]+(:[a-z_]+){1,2}$'),
    description TEXT NOT NULL
);

CREATE TABLE role_capabilities (
    role_id       TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    capability_id TEXT NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, capability_id)
);

CREATE TABLE users (
    id            UUID PRIMARY KEY,
    username      TEXT NOT NULL CHECK (username ~ '^[A-Za-z0-9_.-]{3,32}$'),
    email         TEXT NOT NULL CHECK (email = lower(email) AND email ~ '^[^@\s]+@[^@\s]+$'),
    password_hash TEXT NOT NULL CHECK (password_hash LIKE '$argon2id$%'),
    status        TEXT NOT NULL CHECK (status IN ('active', 'suspended', 'disabled')),
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);
CREATE UNIQUE INDEX uq_users_username_ci ON users (lower(username));
CREATE UNIQUE INDEX uq_users_email ON users (email);

CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    TEXT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    granted_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX idx_user_roles_role ON user_roles (role_id);

-- A session family is one login. Access tokens carry the family id (sid) and
-- are rejected as soon as the family is revoked or expires.
CREATE TABLE session_families (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    revoke_reason TEXT CHECK (revoke_reason IN ('logout', 'logout_all', 'reuse_detected', 'user_disabled', 'roles_changed', 'pruned')),
    user_agent    TEXT NOT NULL DEFAULT '',
    ip_address    TEXT NOT NULL DEFAULT '',
    CHECK (expires_at > created_at),
    CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL))
);
CREATE INDEX idx_session_families_user_active ON session_families (user_id) WHERE revoked_at IS NULL;

-- Each refresh token is single use. Rotation marks used_at and links the
-- successor through replaced_by in the same transaction.
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY,
    family_id   UUID NOT NULL REFERENCES session_families(id) ON DELETE CASCADE,
    generation  INT NOT NULL CHECK (generation >= 1),
    token_hash  CHAR(64) NOT NULL CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    created_at  TIMESTAMPTZ NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    replaced_by UUID REFERENCES refresh_tokens(id) ON DELETE SET NULL,
    CHECK (expires_at > created_at),
    CHECK (replaced_by IS NULL OR used_at IS NOT NULL),
    CONSTRAINT uq_refresh_tokens_family_generation UNIQUE (family_id, generation)
);
CREATE UNIQUE INDEX uq_refresh_tokens_hash ON refresh_tokens (token_hash);

CREATE TABLE audit_log (
    id            UUID PRIMARY KEY,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL,
    details       JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object'),
    created_at    TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_audit_log_resource ON audit_log (resource_type, resource_id, created_at);
CREATE TRIGGER trg_audit_log_append_only BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

-- ---------------------------------------------------------------------------
-- Games, engine artifacts and workers
-- ---------------------------------------------------------------------------

-- A game row registers a game module (games/<id>/manifest.yaml) as available.
CREATE TABLE games (
    id          TEXT PRIMARY KEY CHECK (id ~ '^[a-z][a-z0-9_-]{1,31}$'),
    name        TEXT NOT NULL,
    version     TEXT NOT NULL CHECK (version ~ '^[0-9]+\.[0-9]+\.[0-9]+$'),
    min_players INT NOT NULL CHECK (min_players >= 1),
    max_players INT NOT NULL,
    status      TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL,
    CHECK (max_players >= min_players)
);

-- Engine binaries announced by workers. The digest is what an ExecutionSpec
-- pins; a worker only reserves jobs whose engine digest equals its own.
CREATE TABLE engine_artifacts (
    sha256           CHAR(64) PRIMARY KEY CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    game_id          TEXT NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    game_version     TEXT NOT NULL,
    engine_version   TEXT NOT NULL,
    protocol_version TEXT NOT NULL,
    source_commit    TEXT NOT NULL DEFAULT '',
    first_seen_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE workers (
    id              TEXT PRIMARY KEY CHECK (length(id) BETWEEN 1 AND 128),
    engine_sha256   CHAR(64) NOT NULL REFERENCES engine_artifacts(sha256) ON DELETE RESTRICT,
    sandbox_runtime TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL,
    last_seen_at    TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_workers_engine_seen ON workers (engine_sha256, last_seen_at DESC);

-- ---------------------------------------------------------------------------
-- Agents and submissions
-- ---------------------------------------------------------------------------

CREATE TABLE agents (
    id            UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    game_id       TEXT NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    name          TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 64),
    description   TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 2000),
    status        TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_agents_owner_name UNIQUE (owner_user_id, name),
    CONSTRAINT uq_agents_identity UNIQUE (id, owner_user_id, game_id)
);
CREATE INDEX idx_agents_game ON agents (game_id);

CREATE TABLE submissions (
    id              UUID PRIMARY KEY,
    agent_id        UUID NOT NULL REFERENCES agents(id) ON DELETE RESTRICT,
    version         INT NOT NULL CHECK (version > 0),
    status          TEXT NOT NULL CHECK (status IN ('validating', 'ready', 'rejected', 'disabled')),
    runtime         TEXT NOT NULL CHECK (runtime IN ('python3')),
    entrypoint      TEXT NOT NULL CHECK (entrypoint ~ '^[A-Za-z0-9_][A-Za-z0-9_.-]*\.py$'),
    artifact_key    TEXT NOT NULL CHECK (artifact_key ~ '^submissions/sha256/[0-9a-f]{64}\.py$'),
    artifact_sha256 CHAR(64) NOT NULL CHECK (artifact_sha256 ~ '^[0-9a-f]{64}$'),
    size_bytes      BIGINT NOT NULL CHECK (size_bytes > 0),
    manifest        JSONB NOT NULL CHECK (jsonb_typeof(manifest) = 'object'),
    admission_error TEXT CHECK (admission_error IS NULL OR length(admission_error) <= 1000),
    created_by      UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_submissions_agent_version UNIQUE (agent_id, version),
    CONSTRAINT uq_submissions_identity UNIQUE (id, agent_id),
    CHECK (artifact_key = 'submissions/sha256/' || artifact_sha256 || '.py'),
    CHECK ((status = 'rejected') = (admission_error IS NOT NULL))
);

CREATE FUNCTION agentrix_submissions_immutable() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.agent_id <> OLD.agent_id OR NEW.version <> OLD.version OR NEW.runtime <> OLD.runtime
       OR NEW.entrypoint <> OLD.entrypoint OR NEW.artifact_key <> OLD.artifact_key
       OR NEW.artifact_sha256 <> OLD.artifact_sha256 OR NEW.size_bytes <> OLD.size_bytes
       OR NEW.manifest <> OLD.manifest OR NEW.created_by <> OLD.created_by THEN
        RAISE EXCEPTION 'submission % is immutable', OLD.id USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.status <> NEW.status AND NOT (
        (OLD.status = 'validating' AND NEW.status IN ('ready', 'rejected')) OR
        (OLD.status = 'ready' AND NEW.status = 'disabled')) THEN
        RAISE EXCEPTION 'invalid submission transition % -> %', OLD.status, NEW.status USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_submissions_immutable BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION agentrix_submissions_immutable();
CREATE TRIGGER trg_submissions_no_delete BEFORE DELETE ON submissions
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

-- ---------------------------------------------------------------------------
-- Contests and entries
-- ---------------------------------------------------------------------------

CREATE TABLE contests (
    id             UUID PRIMARY KEY,
    game_id        TEXT NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    name           TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 128),
    description    TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 4000),
    state          TEXT NOT NULL CHECK (state IN ('draft', 'registration_open', 'registration_closed',
                                                  'running', 'finished', 'cancelled', 'archived')),
    starts_at      TIMESTAMPTZ,
    ends_at        TIMESTAMPTZ,
    scoring_policy JSONB NOT NULL CHECK (jsonb_typeof(scoring_policy) = 'object'),
    created_by     UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL,
    CONSTRAINT ck_contests_window CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at),
    CONSTRAINT uq_contests_game UNIQUE (id, game_id)
);
CREATE INDEX idx_contests_state ON contests (state, created_at DESC);

CREATE FUNCTION agentrix_contests_transition() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.state = NEW.state THEN
        IF OLD.state <> 'draft' AND NEW.scoring_policy <> OLD.scoring_policy THEN
            RAISE EXCEPTION 'scoring policy is frozen once a contest leaves draft' USING ERRCODE = 'check_violation';
        END IF;
        IF NEW.game_id <> OLD.game_id THEN
            RAISE EXCEPTION 'contest game cannot change' USING ERRCODE = 'check_violation';
        END IF;
        RETURN NEW;
    END IF;
    IF NOT (
        (OLD.state = 'draft' AND NEW.state IN ('registration_open', 'cancelled')) OR
        (OLD.state = 'registration_open' AND NEW.state IN ('registration_closed', 'cancelled')) OR
        (OLD.state = 'registration_closed' AND NEW.state IN ('running', 'cancelled')) OR
        (OLD.state = 'running' AND NEW.state IN ('finished', 'cancelled')) OR
        (OLD.state = 'finished' AND NEW.state = 'archived') OR
        (OLD.state = 'cancelled' AND NEW.state = 'archived')) THEN
        RAISE EXCEPTION 'invalid contest transition % -> %', OLD.state, NEW.state USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_contests_transition BEFORE UPDATE ON contests
    FOR EACH ROW EXECUTE FUNCTION agentrix_contests_transition();

CREATE TABLE contest_entries (
    id                UUID PRIMARY KEY,
    contest_id        UUID NOT NULL,
    game_id           TEXT NOT NULL,
    agent_id          UUID NOT NULL,
    user_id           UUID NOT NULL,
    submission_id     UUID NOT NULL,
    status            TEXT NOT NULL CHECK (status IN ('enrolled', 'withdrawn', 'disqualified')),
    status_reason     TEXT CHECK (status_reason IS NULL OR length(status_reason) <= 500),
    status_changed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    status_changed_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_contest_entries_agent UNIQUE (contest_id, agent_id),
    CONSTRAINT uq_contest_entries_identity UNIQUE (id, submission_id),
    CONSTRAINT fk_contest_entries_contest FOREIGN KEY (contest_id, game_id)
        REFERENCES contests(id, game_id) ON DELETE RESTRICT,
    CONSTRAINT fk_contest_entries_agent FOREIGN KEY (agent_id, user_id, game_id)
        REFERENCES agents(id, owner_user_id, game_id) ON DELETE RESTRICT,
    CONSTRAINT fk_contest_entries_submission FOREIGN KEY (submission_id, agent_id)
        REFERENCES submissions(id, agent_id) ON DELETE RESTRICT,
    CHECK ((status = 'enrolled') = (status_changed_at IS NULL)),
    CHECK (status = 'enrolled' OR (status_reason IS NOT NULL AND status_changed_by IS NOT NULL))
);
CREATE INDEX idx_contest_entries_user ON contest_entries (user_id);

-- The roster (entries and their locked submission) can only change while
-- registration is open. Withdraw/disqualify are status changes and remain
-- possible until the contest ends.
CREATE FUNCTION agentrix_contest_entries_roster() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    contest_state TEXT;
BEGIN
    SELECT state INTO contest_state FROM contests WHERE id = NEW.contest_id;
    IF TG_OP = 'INSERT' OR NEW.submission_id <> OLD.submission_id THEN
        IF contest_state IS DISTINCT FROM 'registration_open' THEN
            RAISE EXCEPTION 'contest roster is frozen (state %)', contest_state USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    IF TG_OP = 'UPDATE' THEN
        IF NEW.contest_id <> OLD.contest_id OR NEW.agent_id <> OLD.agent_id OR NEW.user_id <> OLD.user_id THEN
            RAISE EXCEPTION 'contest entry identity is immutable' USING ERRCODE = 'check_violation';
        END IF;
        IF OLD.status <> NEW.status AND OLD.status <> 'enrolled' THEN
            RAISE EXCEPTION 'contest entry status % is terminal', OLD.status USING ERRCODE = 'check_violation';
        END IF;
        IF OLD.status <> NEW.status AND contest_state IN ('finished', 'cancelled', 'archived') THEN
            RAISE EXCEPTION 'contest entry cannot change once the contest is %', contest_state USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_contest_entries_roster BEFORE INSERT OR UPDATE ON contest_entries
    FOR EACH ROW EXECUTE FUNCTION agentrix_contest_entries_roster();
CREATE TRIGGER trg_contest_entries_no_delete BEFORE DELETE ON contest_entries
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

-- ---------------------------------------------------------------------------
-- Matches, slots, runs and jobs
-- ---------------------------------------------------------------------------

CREATE TABLE matches (
    id               UUID PRIMARY KEY,
    contest_id       UUID,
    game_id          TEXT NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    mode             TEXT NOT NULL CHECK (mode IN ('competitive', 'exhibition', 'demo')),
    state            TEXT NOT NULL CHECK (state IN ('scheduled', 'queued', 'running', 'finished', 'failed', 'cancelled')),
    seed             BIGINT NOT NULL,
    committed_run_id UUID,
    created_by       UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at       TIMESTAMPTZ NOT NULL,
    updated_at       TIMESTAMPTZ NOT NULL,
    finished_at      TIMESTAMPTZ,
    CONSTRAINT fk_matches_contest FOREIGN KEY (contest_id, game_id)
        REFERENCES contests(id, game_id) ON DELETE RESTRICT,
    CONSTRAINT uq_matches_mode UNIQUE (id, mode),
    CONSTRAINT ck_matches_competitive_contest CHECK (mode <> 'competitive' OR contest_id IS NOT NULL),
    CONSTRAINT ck_matches_committed CHECK ((state = 'finished') = (committed_run_id IS NOT NULL)),
    CONSTRAINT ck_matches_finished_at CHECK ((state = 'finished') = (finished_at IS NOT NULL))
);
CREATE INDEX idx_matches_contest ON matches (contest_id, created_at DESC);
CREATE INDEX idx_matches_state ON matches (state);

CREATE FUNCTION agentrix_matches_transition() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.contest_id IS DISTINCT FROM OLD.contest_id OR NEW.game_id <> OLD.game_id
       OR NEW.mode <> OLD.mode OR NEW.seed <> OLD.seed OR NEW.created_by <> OLD.created_by THEN
        RAISE EXCEPTION 'match identity is immutable' USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.committed_run_id IS NOT NULL AND NEW.committed_run_id IS DISTINCT FROM OLD.committed_run_id THEN
        RAISE EXCEPTION 'committed run cannot change' USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.state <> NEW.state AND NOT (
        (OLD.state = 'scheduled' AND NEW.state IN ('queued', 'cancelled')) OR
        (OLD.state = 'queued' AND NEW.state IN ('running', 'failed', 'cancelled')) OR
        (OLD.state = 'running' AND NEW.state IN ('finished', 'failed')) OR
        (OLD.state = 'failed' AND NEW.state IN ('queued', 'cancelled'))) THEN
        RAISE EXCEPTION 'invalid match transition % -> %', OLD.state, NEW.state USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_matches_transition BEFORE UPDATE ON matches
    FOR EACH ROW EXECUTE FUNCTION agentrix_matches_transition();
CREATE TRIGGER trg_matches_no_delete BEFORE DELETE ON matches
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

CREATE TABLE match_slots (
    id               UUID PRIMARY KEY,
    match_id         UUID NOT NULL,
    match_mode       TEXT NOT NULL,
    slot_index       INT NOT NULL CHECK (slot_index >= 0),
    agent_id         UUID NOT NULL,
    submission_id    UUID NOT NULL,
    contest_entry_id UUID,
    display_name     TEXT NOT NULL CHECK (length(display_name) BETWEEN 1 AND 160),
    created_at       TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_match_slots_index UNIQUE (match_id, slot_index),
    CONSTRAINT uq_match_slots_identity UNIQUE (id, match_id, submission_id),
    CONSTRAINT fk_match_slots_match FOREIGN KEY (match_id, match_mode)
        REFERENCES matches(id, mode) ON DELETE RESTRICT,
    CONSTRAINT fk_match_slots_submission FOREIGN KEY (submission_id, agent_id)
        REFERENCES submissions(id, agent_id) ON DELETE RESTRICT,
    CONSTRAINT fk_match_slots_entry FOREIGN KEY (contest_entry_id, submission_id)
        REFERENCES contest_entries(id, submission_id) ON DELETE RESTRICT,
    CONSTRAINT ck_match_slots_competitive_entry CHECK (match_mode <> 'competitive' OR contest_entry_id IS NOT NULL)
);
CREATE UNIQUE INDEX uq_match_slots_competitive_submission ON match_slots (match_id, submission_id)
    WHERE match_mode = 'competitive';
CREATE TRIGGER trg_match_slots_immutable BEFORE UPDATE OR DELETE ON match_slots
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

CREATE SEQUENCE fencing_token_seq AS BIGINT START WITH 1 MINVALUE 1;

CREATE TABLE match_runs (
    id                  UUID PRIMARY KEY,
    match_id            UUID NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    attempt             INT NOT NULL CHECK (attempt > 0),
    state               TEXT NOT NULL CHECK (state IN ('created', 'reserved', 'running', 'committed',
                                                       'failed', 'timed_out', 'aborted')),
    execution_spec      JSONB NOT NULL CHECK (jsonb_typeof(execution_spec) = 'object'),
    execution_spec_hash CHAR(64) NOT NULL CHECK (execution_spec_hash ~ '^[0-9a-f]{64}$'),
    engine_sha256       CHAR(64) NOT NULL REFERENCES engine_artifacts(sha256) ON DELETE RESTRICT,
    worker_id           TEXT,
    sandbox_runtime     TEXT,
    fencing_token       BIGINT CHECK (fencing_token IS NULL OR fencing_token > 0),
    created_at          TIMESTAMPTZ NOT NULL,
    reserved_at         TIMESTAMPTZ,
    started_at          TIMESTAMPTZ,
    heartbeat_at        TIMESTAMPTZ,
    finished_at         TIMESTAMPTZ,
    error_class         TEXT CHECK (error_class IN ('infrastructure', 'sandbox', 'engine', 'artifact',
                                                    'lease_lost', 'cancelled', 'spec')),
    error_message       TEXT CHECK (error_message IS NULL OR length(error_message) <= 1000),
    CONSTRAINT uq_match_runs_attempt UNIQUE (match_id, attempt),
    CONSTRAINT uq_match_runs_identity UNIQUE (id, match_id),
    CONSTRAINT ck_match_runs_reservation CHECK (
        state IN ('created', 'aborted') OR (worker_id IS NOT NULL AND fencing_token IS NOT NULL AND reserved_at IS NOT NULL)),
    CONSTRAINT ck_match_runs_terminal CHECK (
        (state IN ('committed', 'failed', 'timed_out', 'aborted')) = (finished_at IS NOT NULL)),
    CONSTRAINT ck_match_runs_error CHECK (
        (state IN ('failed', 'timed_out', 'aborted')) = (error_class IS NOT NULL))
);
CREATE UNIQUE INDEX uq_match_runs_committed ON match_runs (match_id) WHERE state = 'committed';

CREATE FUNCTION agentrix_match_runs_transition() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.match_id <> OLD.match_id OR NEW.attempt <> OLD.attempt
       OR NEW.execution_spec <> OLD.execution_spec OR NEW.execution_spec_hash <> OLD.execution_spec_hash
       OR NEW.engine_sha256 <> OLD.engine_sha256 THEN
        RAISE EXCEPTION 'match run identity and execution spec are immutable' USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.state IN ('committed', 'failed', 'timed_out', 'aborted') THEN
        RAISE EXCEPTION 'match run % is terminal (%)', OLD.id, OLD.state USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.fencing_token IS NOT NULL AND NEW.fencing_token IS DISTINCT FROM OLD.fencing_token THEN
        RAISE EXCEPTION 'fencing token of run % cannot change', OLD.id USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.state <> NEW.state AND NOT (
        (OLD.state = 'created' AND NEW.state IN ('reserved', 'aborted')) OR
        (OLD.state = 'reserved' AND NEW.state IN ('running', 'failed', 'timed_out', 'aborted')) OR
        (OLD.state = 'running' AND NEW.state IN ('committed', 'failed', 'timed_out', 'aborted'))) THEN
        RAISE EXCEPTION 'invalid match run transition % -> %', OLD.state, NEW.state USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_match_runs_transition BEFORE UPDATE ON match_runs
    FOR EACH ROW EXECUTE FUNCTION agentrix_match_runs_transition();
CREATE TRIGGER trg_match_runs_no_delete BEFORE DELETE ON match_runs
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

ALTER TABLE matches ADD CONSTRAINT fk_matches_committed_run
    FOREIGN KEY (committed_run_id, id) REFERENCES match_runs(id, match_id) ON DELETE RESTRICT;

-- One job per run. Terminal jobs are kept as history; at most one active job
-- per match.
CREATE TABLE match_jobs (
    id            UUID PRIMARY KEY,
    match_id      UUID NOT NULL,
    run_id        UUID NOT NULL,
    engine_sha256 CHAR(64) NOT NULL REFERENCES engine_artifacts(sha256) ON DELETE RESTRICT,
    state         TEXT NOT NULL CHECK (state IN ('pending', 'reserved', 'completed', 'failed', 'cancelled')),
    available_at  TIMESTAMPTZ NOT NULL,
    reserved_by   TEXT,
    reserved_at   TIMESTAMPTZ,
    lease_until   TIMESTAMPTZ,
    fencing_token BIGINT CHECK (fencing_token IS NULL OR fencing_token > 0),
    last_error    TEXT CHECK (last_error IS NULL OR length(last_error) <= 1000),
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_match_jobs_run UNIQUE (run_id),
    CONSTRAINT fk_match_jobs_run FOREIGN KEY (run_id, match_id) REFERENCES match_runs(id, match_id) ON DELETE RESTRICT,
    CONSTRAINT ck_match_jobs_reserved CHECK (
        state <> 'reserved' OR (reserved_by IS NOT NULL AND reserved_at IS NOT NULL
                                AND lease_until IS NOT NULL AND fencing_token IS NOT NULL))
);
CREATE UNIQUE INDEX uq_match_jobs_active ON match_jobs (match_id) WHERE state IN ('pending', 'reserved');
CREATE INDEX idx_match_jobs_pending ON match_jobs (engine_sha256, available_at, created_at) WHERE state = 'pending';
CREATE INDEX idx_match_jobs_leases ON match_jobs (lease_until) WHERE state = 'reserved';

CREATE FUNCTION agentrix_match_jobs_transition() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.match_id <> OLD.match_id OR NEW.run_id <> OLD.run_id OR NEW.engine_sha256 <> OLD.engine_sha256 THEN
        RAISE EXCEPTION 'match job identity is immutable' USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.state IN ('completed', 'failed', 'cancelled') THEN
        RAISE EXCEPTION 'match job % is terminal (%)', OLD.id, OLD.state USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.state <> NEW.state AND NOT (
        (OLD.state = 'pending' AND NEW.state IN ('reserved', 'cancelled')) OR
        (OLD.state = 'reserved' AND NEW.state IN ('completed', 'failed'))) THEN
        RAISE EXCEPTION 'invalid match job transition % -> %', OLD.state, NEW.state USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_match_jobs_transition BEFORE UPDATE ON match_jobs
    FOR EACH ROW EXECUTE FUNCTION agentrix_match_jobs_transition();
CREATE TRIGGER trg_match_jobs_no_delete BEFORE DELETE ON match_jobs
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

-- ---------------------------------------------------------------------------
-- Results and replays (owned by one run)
-- ---------------------------------------------------------------------------

CREATE TABLE results (
    id            UUID PRIMARY KEY,
    match_run_id  UUID NOT NULL,
    match_id      UUID NOT NULL,
    slot_id       UUID NOT NULL,
    submission_id UUID NOT NULL,
    score         INT NOT NULL,
    rank          INT NOT NULL CHECK (rank >= 1),
    outcome       TEXT NOT NULL CHECK (outcome IN ('win', 'loss', 'draw', 'disqualified')),
    details       JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(details) = 'object'),
    created_at    TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_results_run_slot UNIQUE (match_run_id, slot_id),
    CONSTRAINT fk_results_run FOREIGN KEY (match_run_id, match_id) REFERENCES match_runs(id, match_id) ON DELETE RESTRICT,
    CONSTRAINT fk_results_slot FOREIGN KEY (slot_id, match_id, submission_id)
        REFERENCES match_slots(id, match_id, submission_id) ON DELETE RESTRICT
);
CREATE INDEX idx_results_match ON results (match_id);
CREATE TRIGGER trg_results_immutable BEFORE UPDATE OR DELETE ON results
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

CREATE TABLE replays (
    id                  UUID PRIMARY KEY,
    match_run_id        UUID NOT NULL,
    match_id            UUID NOT NULL,
    format_version      TEXT NOT NULL,
    compression         TEXT NOT NULL CHECK (compression IN ('none', 'gzip')),
    staging_key         TEXT NOT NULL CHECK (staging_key LIKE 'replays/staging/%'),
    storage_key         TEXT NOT NULL CHECK (storage_key LIKE 'replays/published/%'),
    sha256              CHAR(64) NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    size_bytes          BIGINT NOT NULL CHECK (size_bytes > 0),
    frame_count         INT NOT NULL CHECK (frame_count > 0),
    publication_state   TEXT NOT NULL CHECK (publication_state IN ('staging', 'published', 'publish_failed')),
    publish_attempts    INT NOT NULL DEFAULT 0 CHECK (publish_attempts >= 0),
    last_publish_error  TEXT CHECK (last_publish_error IS NULL OR length(last_publish_error) <= 1000),
    next_attempt_at     TIMESTAMPTZ NOT NULL,
    published_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_replays_run UNIQUE (match_run_id),
    CONSTRAINT fk_replays_run FOREIGN KEY (match_run_id, match_id) REFERENCES match_runs(id, match_id) ON DELETE RESTRICT,
    CONSTRAINT ck_replays_published CHECK ((publication_state = 'published') = (published_at IS NOT NULL))
);
CREATE INDEX idx_replays_pending ON replays (next_attempt_at) WHERE publication_state <> 'published';

CREATE FUNCTION agentrix_replays_transition() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.match_run_id <> OLD.match_run_id OR NEW.match_id <> OLD.match_id OR NEW.sha256 <> OLD.sha256
       OR NEW.size_bytes <> OLD.size_bytes OR NEW.storage_key <> OLD.storage_key OR NEW.staging_key <> OLD.staging_key
       OR NEW.compression <> OLD.compression OR NEW.format_version <> OLD.format_version OR NEW.frame_count <> OLD.frame_count THEN
        RAISE EXCEPTION 'replay identity and digest are immutable' USING ERRCODE = 'check_violation';
    END IF;
    IF OLD.publication_state = 'published' THEN
        RAISE EXCEPTION 'published replay % is immutable', OLD.id USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER trg_replays_transition BEFORE UPDATE ON replays
    FOR EACH ROW EXECUTE FUNCTION agentrix_replays_transition();
CREATE TRIGGER trg_replays_no_delete BEFORE DELETE ON replays
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

-- ---------------------------------------------------------------------------
-- Rankings: projection of committed runs + immutable published snapshots
-- ---------------------------------------------------------------------------

-- Every commit of a contest match bumps dirty_version in the commit
-- transaction; a recomputation stores the version it covered.
CREATE TABLE contest_ranking_state (
    contest_id          UUID PRIMARY KEY REFERENCES contests(id) ON DELETE RESTRICT,
    dirty_version       BIGINT NOT NULL DEFAULT 0 CHECK (dirty_version >= 0),
    computed_version    BIGINT NOT NULL DEFAULT 0 CHECK (computed_version >= 0),
    applied_runs_count  INT NOT NULL DEFAULT 0 CHECK (applied_runs_count >= 0),
    applied_runs_digest CHAR(64) NOT NULL DEFAULT repeat('0', 64),
    computed_at         TIMESTAMPTZ,
    CHECK (computed_version <= dirty_version)
);

CREATE TABLE rankings (
    contest_id        UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    entry_id          UUID NOT NULL REFERENCES contest_entries(id) ON DELETE RESTRICT,
    agent_id          UUID NOT NULL,
    user_id           UUID NOT NULL,
    rank              INT NOT NULL CHECK (rank >= 1),
    points            INT NOT NULL,
    matches_played    INT NOT NULL CHECK (matches_played >= 0),
    wins              INT NOT NULL CHECK (wins >= 0),
    draws             INT NOT NULL CHECK (draws >= 0),
    losses            INT NOT NULL CHECK (losses >= 0),
    disqualifications INT NOT NULL CHECK (disqualifications >= 0),
    score_for         INT NOT NULL,
    score_against     INT NOT NULL,
    PRIMARY KEY (contest_id, entry_id),
    CHECK (wins + draws + losses + disqualifications = matches_played)
);

CREATE TABLE ranking_applied_runs (
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    run_id     UUID NOT NULL REFERENCES match_runs(id) ON DELETE RESTRICT,
    match_id   UUID NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    PRIMARY KEY (contest_id, run_id),
    CONSTRAINT uq_ranking_applied_runs_match UNIQUE (contest_id, match_id)
);

CREATE TABLE ranking_snapshots (
    id                  UUID PRIMARY KEY,
    contest_id          UUID NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    version             INT NOT NULL CHECK (version > 0),
    rankings            JSONB NOT NULL CHECK (jsonb_typeof(rankings) = 'array'),
    rankings_sha256     CHAR(64) NOT NULL CHECK (rankings_sha256 ~ '^[0-9a-f]{64}$'),
    applied_runs_digest CHAR(64) NOT NULL,
    applied_runs_count  INT NOT NULL CHECK (applied_runs_count >= 0),
    published_by        UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at        TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_ranking_snapshots_version UNIQUE (contest_id, version)
);
CREATE TRIGGER trg_ranking_snapshots_immutable BEFORE UPDATE OR DELETE ON ranking_snapshots
    FOR EACH ROW EXECUTE FUNCTION agentrix_forbid_mutation();

-- ---------------------------------------------------------------------------
-- Idempotency (scoped by actor + operation + key)
-- ---------------------------------------------------------------------------

CREATE TABLE idempotency_keys (
    actor_user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operation       TEXT NOT NULL,
    key             TEXT NOT NULL CHECK (key ~ '^[A-Za-z0-9_.:-]{8,128}$'),
    request_hash    CHAR(64) NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    response_status INT NOT NULL CHECK (response_status BETWEEN 200 AND 299),
    response_body   JSONB NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (actor_user_id, operation, key),
    CHECK (expires_at > created_at)
);
CREATE INDEX idx_idempotency_keys_expiry ON idempotency_keys (expires_at);

-- ---------------------------------------------------------------------------
-- RBAC policy reference data
-- ---------------------------------------------------------------------------

INSERT INTO roles (id, description) VALUES
    ('admin', 'Platform administrator'),
    ('organizer', 'Contest organizer'),
    ('player', 'Competitor and bot author'),
    ('referee', 'Match official'),
    ('spectator', 'Read-only audience; also the capabilities of anonymous visitors');

INSERT INTO capabilities (id, description) VALUES
    ('users:read:own', 'Read own account'),
    ('users:read:any', 'Read any account'),
    ('users:update:own', 'Update own account'),
    ('users:update:any', 'Change the status of any account'),
    ('users:roles:manage', 'Replace the roles of an account'),
    ('agents:create', 'Create agents'),
    ('agents:read:own', 'Read own agents'),
    ('agents:read:any', 'Read any agent'),
    ('agents:update:own', 'Update own agents'),
    ('agents:update:any', 'Update any agent'),
    ('submissions:create:own', 'Upload submissions for own agents'),
    ('submissions:read:own', 'Read submissions of own agents'),
    ('submissions:read:any', 'Read any submission'),
    ('contests:view', 'View public contests'),
    ('contests:manage', 'Create contests and drive their state machine'),
    ('entries:create:own', 'Enroll, re-lock and withdraw own agents'),
    ('entries:manage:any', 'Enroll any agent and disqualify entries'),
    ('matches:view', 'View matches, slots and runs'),
    ('matches:create', 'Create matches'),
    ('matches:run', 'Schedule match runs'),
    ('matches:cancel', 'Cancel matches'),
    ('rankings:view', 'View rankings and published snapshots'),
    ('rankings:publish', 'Recalculate rankings and publish snapshots'),
    ('replays:view', 'View published replays'),
    ('admin:access', 'Access administrative readiness and tooling');

INSERT INTO role_capabilities (role_id, capability_id)
SELECT 'admin', id FROM capabilities;

INSERT INTO role_capabilities (role_id, capability_id) VALUES
    ('organizer', 'users:read:own'), ('organizer', 'users:update:own'), ('organizer', 'users:read:any'),
    ('organizer', 'agents:read:any'), ('organizer', 'submissions:read:any'),
    ('organizer', 'contests:view'), ('organizer', 'contests:manage'),
    ('organizer', 'entries:manage:any'),
    ('organizer', 'matches:view'), ('organizer', 'matches:create'), ('organizer', 'matches:run'), ('organizer', 'matches:cancel'),
    ('organizer', 'rankings:view'), ('organizer', 'rankings:publish'),
    ('organizer', 'replays:view'), ('organizer', 'admin:access'),

    ('player', 'users:read:own'), ('player', 'users:update:own'),
    ('player', 'agents:create'), ('player', 'agents:read:own'), ('player', 'agents:update:own'),
    ('player', 'submissions:create:own'), ('player', 'submissions:read:own'),
    ('player', 'entries:create:own'),
    ('player', 'contests:view'), ('player', 'matches:view'), ('player', 'rankings:view'), ('player', 'replays:view'),

    ('referee', 'users:read:own'), ('referee', 'users:update:own'),
    ('referee', 'contests:view'), ('referee', 'matches:view'), ('referee', 'matches:run'),
    ('referee', 'rankings:view'), ('referee', 'replays:view'),

    ('spectator', 'contests:view'), ('spectator', 'matches:view'), ('spectator', 'rankings:view'), ('spectator', 'replays:view');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS idempotency_keys, ranking_snapshots, ranking_applied_runs, rankings, contest_ranking_state,
    replays, results, match_jobs CASCADE;
ALTER TABLE IF EXISTS matches DROP CONSTRAINT IF EXISTS fk_matches_committed_run;
DROP TABLE IF EXISTS match_runs, match_slots, matches, contest_entries, contests, submissions, agents,
    workers, engine_artifacts, games, audit_log, refresh_tokens, session_families, user_roles, users,
    role_capabilities, capabilities, roles CASCADE;
DROP SEQUENCE IF EXISTS fencing_token_seq;
DROP FUNCTION IF EXISTS agentrix_forbid_mutation, agentrix_submissions_immutable, agentrix_contests_transition,
    agentrix_contest_entries_roster, agentrix_matches_transition, agentrix_match_runs_transition,
    agentrix_match_jobs_transition, agentrix_replays_transition;
-- +goose StatementEnd
