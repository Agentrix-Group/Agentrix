-- +goose Up
-- +goose StatementBegin

-- 1. Roles & Permissions (RBAC)
CREATE TABLE IF NOT EXISTS roles (
    id VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS permissions (
    id VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(64) NOT NULL,
    permission_id VARCHAR(64) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- Seed canonical roles and permissions
INSERT INTO roles (id, description, active) VALUES
('admin', 'System Administrator', TRUE),
('organizer', 'Contest and tournament organizer', TRUE),
('player', 'Competitor and bot author', TRUE),
('participant', 'Competitor and bot author (legacy alias)', TRUE),
('referee', 'Tournament official and referee', TRUE),
('spectator', 'Public viewer and audience', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, description, active) VALUES
('read', 'Read system data and resources', TRUE),
('write', 'Modify and update resources', TRUE),
('admin', 'System administration capabilities', TRUE),
('execute-match', 'Run and schedule matches', TRUE),
('submit-agent', 'Upload and manage bot submissions', TRUE),
('create-contest', 'Create and configure tournaments', TRUE),
('manage-users', 'Manage user accounts and role assignments', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('admin', 'read', TRUE),
('admin', 'write', TRUE),
('admin', 'admin', TRUE),
('admin', 'execute-match', TRUE),
('admin', 'submit-agent', TRUE),
('admin', 'create-contest', TRUE),
('admin', 'manage-users', TRUE),
('organizer', 'read', TRUE),
('organizer', 'write', TRUE),
('organizer', 'execute-match', TRUE),
('organizer', 'submit-agent', TRUE),
('organizer', 'create-contest', TRUE),
('player', 'read', TRUE),
('player', 'write', TRUE),
('player', 'submit-agent', TRUE),
('player', 'execute-match', TRUE),
('participant', 'read', TRUE),
('participant', 'write', TRUE),
('participant', 'submit-agent', TRUE),
('participant', 'execute-match', TRUE),
('referee', 'read', TRUE),
('referee', 'execute-match', TRUE),
('spectator', 'read', TRUE)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 2. Users (Canonical Identity)
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

-- 3. Categories
CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(64) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT TRUE
);

-- 4. Games
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

-- 5. Contests
CREATE TABLE IF NOT EXISTS contests (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    game_id VARCHAR(64) REFERENCES games(id) ON DELETE SET NULL,
    category_id VARCHAR(64) REFERENCES categories(id) ON DELETE SET NULL,
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    status VARCHAR(32) DEFAULT 'upcoming',
    active BOOLEAN DEFAULT TRUE,
    starts_at TIMESTAMPTZ NULL,
    ends_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contests_active_state ON contests(active, state);
CREATE INDEX IF NOT EXISTS idx_contests_game_id ON contests(game_id);
CREATE INDEX IF NOT EXISTS idx_contests_created_at ON contests(created_at);

-- 6. Agents
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

-- 7. Submissions
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

-- 8. Contest Entries (Tournament Admission)
CREATE TABLE IF NOT EXISTS contest_entries (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    agent_id VARCHAR(64) NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'enrolled'
        CHECK (status IN ('enrolled', 'disqualified', 'withdrawn')),
    enrolled_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_contest_entries_agent UNIQUE (contest_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_contest_entries_contest ON contest_entries(contest_id);
CREATE INDEX IF NOT EXISTS idx_contest_entries_user ON contest_entries(user_id);

-- 9. Rankings (Leaderboard & Match Scores)
CREATE TABLE IF NOT EXISTS rankings (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    agent_id VARCHAR(64) NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score INT DEFAULT 0,
    matches_played INT DEFAULT 0,
    wins INT DEFAULT 0,
    losses INT DEFAULT 0,
    draws INT DEFAULT 0,
    "rank" INT DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_contest_agent UNIQUE (contest_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_rankings_user_id ON rankings(user_id);
CREATE INDEX IF NOT EXISTS idx_rankings_contest_score ON rankings(contest_id, score DESC);

-- 10. Matches
CREATE TABLE IF NOT EXISTS matches (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) REFERENCES contests(id) ON DELETE SET NULL,
    game_id VARCHAR(64) NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    run_id VARCHAR(64),
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

-- 11. Results
CREATE TABLE IF NOT EXISTS results (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    submission_id VARCHAR(64) NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    score INT DEFAULT 0,
    "rank" INT DEFAULT 1,
    status VARCHAR(32) DEFAULT 'finished',
    details TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_results_match_id ON results(match_id);

-- 12. Replays
CREATE TABLE IF NOT EXISTS replays (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    file_path VARCHAR(255) NOT NULL,
    duration_ticks INT DEFAULT 0,
    summary TEXT,
    sha256 VARCHAR(64),
    size_bytes BIGINT DEFAULT 0,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 13. Match Jobs (Authoritative PostgreSQL Queue)
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

-- 14. Match Runs (Authoritative Worker Execution & Fencing)
CREATE TABLE IF NOT EXISTS match_runs (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    worker_id VARCHAR(128) NOT NULL,
    fencing_token BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'committed', 'aborted', 'superseded')),
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

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS match_runs CASCADE;
DROP TABLE IF EXISTS match_jobs CASCADE;
DROP TABLE IF EXISTS replays CASCADE;
DROP TABLE IF EXISTS results CASCADE;
DROP TABLE IF EXISTS matches CASCADE;
DROP TABLE IF EXISTS rankings CASCADE;
DROP TABLE IF EXISTS contest_entries CASCADE;
DROP TABLE IF EXISTS submissions CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS contests CASCADE;
DROP TABLE IF EXISTS games CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
-- +goose StatementEnd
