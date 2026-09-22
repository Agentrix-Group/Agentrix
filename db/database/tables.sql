-- PostgreSQL DDL Tables for Agentrix platform
-- Fully canonical schema reflecting complete entity lifecycle, RBAC, execution chain, and rankings.

-- ============================================================================
-- 1. Base Tables (No Foreign Keys)
-- ============================================================================

-- Table: roles
CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

-- Table: permissions
CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

-- Table: categories
CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

-- Table: games
CREATE TABLE IF NOT EXISTS games (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    manifest_path VARCHAR(255),
    min_players INT DEFAULT 2,
    max_players INT DEFAULT 4,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Table: idempotency_keys
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(255) PRIMARY KEY,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    response_body JSONB NOT NULL,
    status_code INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_idempotency_keys_resource ON idempotency_keys(resource_type, resource_id);

-- ============================================================================
-- 2. User Identity, RBAC & Sessions
-- ============================================================================

-- Table: role_permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(64) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id VARCHAR(64) NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    active BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (role_id, permission_id)
);

-- Table: users
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(64) PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(128) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    role_id VARCHAR(64) NOT NULL REFERENCES roles(id),
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);

-- Table: user_roles (Many-to-Many RBAC)
CREATE TABLE IF NOT EXISTS user_roles (
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id VARCHAR(64) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

-- Table: sessions (Refresh Token and Session Tracking)
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_id VARCHAR(64) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    user_agent VARCHAR(255) DEFAULT '',
    ip_address VARCHAR(45) DEFAULT '',
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_family_id ON sessions(family_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(user_id, is_revoked, expires_at);

-- ============================================================================
-- 3. Contests, Agents & Submissions
-- ============================================================================

-- Table: contests
CREATE TABLE IF NOT EXISTS contests (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    game_id VARCHAR(64) REFERENCES games(id) ON DELETE SET NULL,
    category_id VARCHAR(64) REFERENCES categories(id) ON DELETE SET NULL,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    status VARCHAR(32) DEFAULT 'upcoming',
    scoring_policy JSONB NOT NULL DEFAULT '{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}'::jsonb,
    active BOOLEAN DEFAULT TRUE,
    starts_at TIMESTAMPTZ NULL,
    ends_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contests_active_state ON contests(active, state);
CREATE INDEX IF NOT EXISTS idx_contests_game_id ON contests(game_id);
CREATE INDEX IF NOT EXISTS idx_contests_created_at ON contests(created_at);

-- Table: agents
CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR(64) PRIMARY KEY,
    owner_user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_id VARCHAR(64) NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agents_owner_user_id ON agents(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_agents_game_id ON agents(game_id);

-- Table: submissions
CREATE TABLE IF NOT EXISTS submissions (
    id VARCHAR(64) PRIMARY KEY,
    agent_id VARCHAR(64) NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    code_path VARCHAR(255),
    language VARCHAR(32) DEFAULT 'python',
    status VARCHAR(32) DEFAULT 'ready',
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_submissions_agent_id ON submissions(agent_id);

-- Table: contest_entries
CREATE TABLE IF NOT EXISTS contest_entries (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    agent_id VARCHAR(64) NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    submission_id VARCHAR(64) NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL DEFAULT 'enrolled'
        CHECK (status IN ('enrolled', 'active', 'disqualified', 'withdrawn')),
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_contest_entries_agent UNIQUE (contest_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_contest_entries_contest ON contest_entries(contest_id);
CREATE INDEX IF NOT EXISTS idx_contest_entries_user ON contest_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_contest_entries_submission_id ON contest_entries(submission_id);

-- ============================================================================
-- 4. Matches, Execution Chain & Worker Queue
-- ============================================================================

-- Table: matches
CREATE TABLE IF NOT EXISTS matches (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) REFERENCES contests(id) ON DELETE SET NULL,
    game_id VARCHAR(64) NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    run_id VARCHAR(64),
    committed_run_id VARCHAR(64),
    game_version VARCHAR(32),
    engine_digest VARCHAR(64),
    config_hash VARCHAR(64),
    status VARCHAR(32) DEFAULT 'pending',
    seed BIGINT DEFAULT 0,
    replay_id VARCHAR(64),
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_matches_contest_id ON matches(contest_id);
CREATE INDEX IF NOT EXISTS idx_matches_status ON matches(status);
CREATE INDEX IF NOT EXISTS idx_matches_committed_run_id ON matches(committed_run_id);

-- Table: match_slots (Ordered, Immutable Slots per Match)
CREATE TABLE IF NOT EXISTS match_slots (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    slot_index INT NOT NULL,
    contest_entry_id VARCHAR(64) REFERENCES contest_entries(id) ON DELETE SET NULL,
    submission_id VARCHAR(64) NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
    agent_name VARCHAR(128) NOT NULL DEFAULT '',
    username VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_match_slots_index UNIQUE (match_id, slot_index)
);

CREATE INDEX IF NOT EXISTS idx_match_slots_match_id ON match_slots(match_id);
CREATE INDEX IF NOT EXISTS idx_match_slots_submission_id ON match_slots(submission_id);

-- Table: match_jobs (Authoritative PostgreSQL Queue)
CREATE TABLE IF NOT EXISTS match_jobs (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL UNIQUE REFERENCES matches(id) ON DELETE CASCADE,
    run_id VARCHAR(64),
    contest_id VARCHAR(64) REFERENCES contests(id) ON DELETE SET NULL,
    game_id VARCHAR(64) NOT NULL DEFAULT 'starfighter' REFERENCES games(id) ON DELETE CASCADE,
    submission_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    seed BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'reserved', 'completed', 'failed')),
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    fencing_token BIGINT NOT NULL DEFAULT 0,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reserved_at TIMESTAMPTZ,
    reserved_by VARCHAR(128),
    lease_until TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_jobs_reservation
    ON match_jobs(status, available_at, created_at);

CREATE INDEX IF NOT EXISTS idx_match_jobs_lease_recovery
    ON match_jobs(status, lease_until)
    WHERE status = 'reserved';

-- Table: match_runs (Authoritative Worker Execution & Fencing)
CREATE TABLE IF NOT EXISTS match_runs (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    worker_id VARCHAR(128) NOT NULL DEFAULT 'unassigned',
    fencing_token BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'created'
        CHECK (status IN ('created', 'dispatching', 'running', 'completed', 'committed', 'failed', 'aborted', 'timed_out', 'superseded')),
    execution_spec JSONB,
    attempt INT NOT NULL DEFAULT 1,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    heartbeat_at TIMESTAMPTZ,
    last_error TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_match_runs_committed
    ON match_runs(match_id)
    WHERE status = 'committed';

CREATE INDEX IF NOT EXISTS idx_match_runs_match_token
    ON match_runs(match_id, fencing_token);

-- Circular FK resolution: matches.committed_run_id -> match_runs(id)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'fk_matches_committed_run' AND table_name = 'matches'
    ) THEN
        ALTER TABLE matches ADD CONSTRAINT fk_matches_committed_run
            FOREIGN KEY (committed_run_id) REFERENCES match_runs(id) ON DELETE SET NULL;
    END IF;
END $$;

-- ============================================================================
-- 5. Rankings, Results & Replays
-- ============================================================================

-- Table: ranking_applied_runs (Idempotency tracking for match result integration)
CREATE TABLE IF NOT EXISTS ranking_applied_runs (
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    run_id VARCHAR(64) NOT NULL REFERENCES match_runs(id) ON DELETE CASCADE,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (contest_id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_ranking_applied_runs_contest ON ranking_applied_runs(contest_id);
CREATE INDEX IF NOT EXISTS idx_ranking_applied_runs_match ON ranking_applied_runs(match_id);

-- Table: rankings (Deterministic leaderboards and tiebreaker statistics)
CREATE TABLE IF NOT EXISTS rankings (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    agent_id VARCHAR(64) NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score INT DEFAULT 0,
    points INT NOT NULL DEFAULT 0,
    matches_played INT DEFAULT 0,
    wins INT DEFAULT 0,
    losses INT DEFAULT 0,
    draws INT DEFAULT 0,
    disqualifications INT NOT NULL DEFAULT 0,
    tiebreaker_score DOUBLE PRECISION NOT NULL DEFAULT 0,
    "rank" INT DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_contest_agent UNIQUE (contest_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_rankings_user_id ON rankings(user_id);
CREATE INDEX IF NOT EXISTS idx_rankings_contest_score ON rankings(contest_id, score DESC);

-- Table: contest_rankings_snapshots (Explicit published versions)
CREATE TABLE IF NOT EXISTS contest_rankings_snapshots (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    version INT NOT NULL,
    snapshot_data JSONB NOT NULL,
    published_by VARCHAR(64) REFERENCES users(id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_contest_rankings_version UNIQUE (contest_id, version)
);

CREATE INDEX IF NOT EXISTS idx_contest_rankings_snapshots_lookup 
    ON contest_rankings_snapshots(contest_id, version DESC);

-- Table: results
CREATE TABLE IF NOT EXISTS results (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    match_run_id VARCHAR(64) REFERENCES match_runs(id) ON DELETE RESTRICT,
    slot_id VARCHAR(64) REFERENCES match_slots(id) ON DELETE SET NULL,
    submission_id VARCHAR(64) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    score INT DEFAULT 0,
    "rank" INT DEFAULT 1,
    status VARCHAR(32) DEFAULT 'finished',
    details TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_results_match_id ON results(match_id);
CREATE INDEX IF NOT EXISTS idx_results_match_run_id ON results(match_run_id);
CREATE INDEX IF NOT EXISTS idx_results_slot_id ON results(slot_id);

-- Table: replays
CREATE TABLE IF NOT EXISTS replays (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    match_run_id VARCHAR(64) REFERENCES match_runs(id) ON DELETE RESTRICT,
    file_path VARCHAR(255) NOT NULL,
    duration_ticks INT DEFAULT 0,
    summary TEXT,
    sha256 VARCHAR(64),
    size_bytes BIGINT DEFAULT 0,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_replays_match_id ON replays(match_id);
CREATE INDEX IF NOT EXISTS idx_replays_match_run_id ON replays(match_run_id);
