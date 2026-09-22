-- ==============================================================================
-- Consolidated Agentrix Database Initialization & Seeds
-- Generated from modular schema files in db/database/ and db/data/
-- ==============================================================================
-- PostgreSQL user and role configuration for Agentrix
-- Creates the agentrix role if it does not already exist, and grants database creation privilege.

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'agentrix') THEN
        CREATE ROLE agentrix WITH LOGIN PASSWORD 'agentrix';
    ELSE
        ALTER ROLE agentrix WITH LOGIN PASSWORD 'agentrix';
    END IF;
END $$;

ALTER ROLE agentrix CREATEDB;

-- PostgreSQL database initialization for Agentrix
-- Drops and recreates the agentrix database, assigning ownership to the agentrix role.

DROP DATABASE IF EXISTS agentrix;
CREATE DATABASE agentrix WITH OWNER agentrix;

GRANT ALL PRIVILEGES ON DATABASE agentrix TO agentrix;

\c agentrix

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

-- PostgreSQL Views for Agentrix platform
-- Canonical filtered views for active and valid system records.

CREATE OR REPLACE VIEW roles_active AS
    SELECT * FROM roles WHERE active = TRUE;

CREATE OR REPLACE VIEW permissions_active AS
    SELECT * FROM permissions WHERE active = TRUE;

CREATE OR REPLACE VIEW role_permissions_active AS
    SELECT * FROM role_permissions WHERE active = TRUE;

CREATE OR REPLACE VIEW users_active AS
    SELECT * FROM users WHERE active = TRUE;

CREATE OR REPLACE VIEW categories_active AS
    SELECT * FROM categories WHERE active = TRUE;

CREATE OR REPLACE VIEW games_active AS
    SELECT * FROM games WHERE active = TRUE;

CREATE OR REPLACE VIEW contests_active AS
    SELECT * FROM contests WHERE active = TRUE;

CREATE OR REPLACE VIEW agents_active AS
    SELECT * FROM agents WHERE active = TRUE;

CREATE OR REPLACE VIEW submissions_active AS
    SELECT * FROM submissions WHERE active = TRUE;

CREATE OR REPLACE VIEW contest_entries_active AS
    SELECT * FROM contest_entries WHERE status != 'disqualified';

CREATE OR REPLACE VIEW matches_active AS
    SELECT * FROM matches WHERE active = TRUE;

CREATE OR REPLACE VIEW replays_active AS
    SELECT * FROM replays WHERE active = TRUE;


-- Data: roles.sql
-- Seed: roles
-- Defines canonical platform roles for RBAC.

INSERT INTO roles (id, description, active) VALUES
('admin', 'System Administrator', TRUE),
('organizer', 'Contest and tournament organizer', TRUE),
('player', 'Competitor and bot author', TRUE),
('participant', 'Competitor and bot author (legacy alias)', TRUE),
('referee', 'Tournament official and referee', TRUE),
('spectator', 'Public viewer and audience', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    description = EXCLUDED.description, 
    active = EXCLUDED.active;

-- Data: permissions.sql
-- Seed: permissions
-- Defines baseline and fine-grained RBAC capabilities for Agentrix.

INSERT INTO permissions (id, description, active) VALUES
-- Baseline permissions
('read', 'Read system data and resources', TRUE),
('write', 'Modify and update resources', TRUE),
('admin', 'System administration capabilities', TRUE),
('execute-match', 'Run and schedule matches', TRUE),
('submit-agent', 'Upload and manage bot submissions', TRUE),
('create-contest', 'Create and configure tournaments', TRUE),
('manage-users', 'Manage user accounts and role assignments', TRUE),

-- Users namespace
('users:read:own', 'Read own user profile and account details', TRUE),
('users:read:any', 'Read any user profile and details', TRUE),
('users:update:own', 'Update own user account', TRUE),
('users:update:any', 'Update any user profile, status, or role', TRUE),

-- Agents namespace
('agents:create', 'Create and register new agents', TRUE),
('agents:read:own', 'Read own registered agents', TRUE),
('agents:read:any', 'Read any registered agent', TRUE),

-- Submissions namespace
('submissions:create:own', 'Upload and register agent code submissions', TRUE),
('submissions:read:own', 'Read own agent submissions and status', TRUE),
('submissions:read:any', 'Read any agent submission', TRUE),

-- Contests namespace
('contests:manage', 'Create, configure, and close contests', TRUE),
('contests:view', 'View contest information', TRUE),
('contests:enroll', 'Enroll agents in contests', TRUE),
('contests:create', 'Create contests (alias)', TRUE),

-- Contest entries namespace
('entries:create:own', 'Enroll own agents in contests', TRUE),
('entries:manage:any', 'Manage contest entries and statuses', TRUE),

-- Matches namespace
('matches:create', 'Schedule and configure matches', TRUE),
('matches:run', 'Trigger execution of matches', TRUE),
('matches:cancel', 'Cancel scheduled matches', TRUE),
('matches:view', 'View public match details and execution states', TRUE),
('matches:schedule', 'Schedule matches (alias)', TRUE),

-- Rankings namespace
('rankings:publish', 'Publish and freeze leaderboard rankings', TRUE),
('rankings:view', 'View public contest rankings', TRUE),

-- Replays & UI namespace
('replays:view', 'View public match replay streams and state', TRUE),
('admin:access', 'Access administrative control panels and tooling', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    description = EXCLUDED.description, 
    active = EXCLUDED.active;

-- Data: role_permissions.sql
-- Seed: role_permissions
-- Associates canonical permissions with each platform role.

-- 1. Admin: full access to all capabilities
INSERT INTO role_permissions (role_id, permission_id, active)
SELECT 'admin', p.id, TRUE
FROM permissions p
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- 2. Organizer: tournament management, match execution, view all submissions/agents
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('organizer', 'read', TRUE),
('organizer', 'write', TRUE),
('organizer', 'execute-match', TRUE),
('organizer', 'submit-agent', TRUE),
('organizer', 'create-contest', TRUE),
('organizer', 'contests:manage', TRUE),
('organizer', 'contests:create', TRUE),
('organizer', 'contests:view', TRUE),
('organizer', 'contests:enroll', TRUE),
('organizer', 'entries:manage:any', TRUE),
('organizer', 'entries:create:own', TRUE),
('organizer', 'matches:create', TRUE),
('organizer', 'matches:run', TRUE),
('organizer', 'matches:cancel', TRUE),
('organizer', 'matches:view', TRUE),
('organizer', 'matches:schedule', TRUE),
('organizer', 'rankings:publish', TRUE),
('organizer', 'rankings:view', TRUE),
('organizer', 'replays:view', TRUE),
('organizer', 'admin:access', TRUE),
('organizer', 'agents:read:any', TRUE),
('organizer', 'submissions:read:any', TRUE),
('organizer', 'users:read:any', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- 3. Player: author bots, enroll in tournaments, inspect own agents/submissions
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('player', 'read', TRUE),
('player', 'write', TRUE),
('player', 'submit-agent', TRUE),
('player', 'execute-match', TRUE),
('player', 'users:read:own', TRUE),
('player', 'users:update:own', TRUE),
('player', 'agents:create', TRUE),
('player', 'agents:read:own', TRUE),
('player', 'submissions:create:own', TRUE),
('player', 'submissions:read:own', TRUE),
('player', 'entries:create:own', TRUE),
('player', 'contests:enroll', TRUE),
('player', 'contests:view', TRUE),
('player', 'matches:view', TRUE),
('player', 'rankings:view', TRUE),
('player', 'replays:view', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- 4. Participant (Legacy alias for Player)
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('participant', 'read', TRUE),
('participant', 'write', TRUE),
('participant', 'submit-agent', TRUE),
('participant', 'execute-match', TRUE),
('participant', 'users:read:own', TRUE),
('participant', 'users:update:own', TRUE),
('participant', 'agents:create', TRUE),
('participant', 'agents:read:own', TRUE),
('participant', 'submissions:create:own', TRUE),
('participant', 'submissions:read:own', TRUE),
('participant', 'entries:create:own', TRUE),
('participant', 'contests:enroll', TRUE),
('participant', 'contests:view', TRUE),
('participant', 'matches:view', TRUE),
('participant', 'rankings:view', TRUE),
('participant', 'replays:view', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- 5. Referee: tournament matches review and execution
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('referee', 'read', TRUE),
('referee', 'execute-match', TRUE),
('referee', 'matches:view', TRUE),
('referee', 'matches:run', TRUE),
('referee', 'rankings:view', TRUE),
('referee', 'replays:view', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- 6. Spectator: public view only
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('spectator', 'read', TRUE),
('spectator', 'contests:view', TRUE),
('spectator', 'matches:view', TRUE),
('spectator', 'rankings:view', TRUE),
('spectator', 'replays:view', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- Data: users.sql
-- Seed: users
-- Default system accounts with SHA-512 hashed credentials.
-- Passwords:
--   admin: admin123
--   pilot_alpha: pilot123
--   pilot_beta: pilot123

INSERT INTO users (id, username, email, password, role_id, active) VALUES
('usr-admin-001', 'admin', 'admin@agentrix.local', '7fcf4ba391c48784edde599889d6e3f1e47a27db36ecc050cc92f259bfac38afad2c68a1ae804d77075e8fb722503f3eca2b2c1006ee6f6c7b7628cb45fffd1d', 'admin', TRUE),
('usr-pilot-001', 'pilot_alpha', 'pilot_alpha@agentrix.local', '5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27', 'player', TRUE),
('usr-pilot-002', 'pilot_beta', 'pilot_beta@agentrix.local', '5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27', 'player', TRUE)
ON CONFLICT (username) DO UPDATE SET 
    email = EXCLUDED.email,
    password = EXCLUDED.password,
    role_id = EXCLUDED.role_id, 
    active = EXCLUDED.active;

-- Data: user_roles.sql
-- Seed: user_roles
-- Assigns roles to default users in the many-to-many RBAC table.

INSERT INTO user_roles (user_id, role_id)
SELECT id, 'admin' FROM users WHERE username = 'admin'
UNION ALL
SELECT id, 'player' FROM users WHERE username = 'pilot_alpha'
UNION ALL
SELECT id, 'player' FROM users WHERE username = 'pilot_beta'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Data: categories.sql
-- Seed: categories
-- Initial contest categories.

INSERT INTO categories (id, description, active) VALUES
('ai-challenge', 'Artificial Intelligence & Algorithmic Bots', TRUE),
('battle-royale', 'Deterministic Starfighter Duel', TRUE),
('heuristics', 'Heuristic Strategy & Optimization', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    description = EXCLUDED.description, 
    active = EXCLUDED.active;

-- Data: games.sql
-- Seed: games
-- Registers the MVP canonical game: Starfighter Arena.

INSERT INTO games (id, name, description, manifest_path, min_players, max_players, active) VALUES
('starfighter', 'Starfighter Arena', 'Deterministic duel between two Python-controlled starfighters.', 'games/starfighter/manifest.yaml', 2, 2, TRUE)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    description = EXCLUDED.description, 
    manifest_path = EXCLUDED.manifest_path,
    min_players = EXCLUDED.min_players,
    max_players = EXCLUDED.max_players,
    active = EXCLUDED.active;

-- Data: contests.sql
-- Seed: contests
-- Initial tournament: Starfighter Championship 2026.

INSERT INTO contests (id, name, description, game_id, category_id, state, status, scoring_policy, active, starts_at, ends_at) VALUES
('starfighter-cup-2026', 'Starfighter Championship 2026', 'Tournament for Python bots controlling starfighters.', 'starfighter', 'battle-royale', 'registration_open', 'upcoming', '{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}'::jsonb, TRUE, NOW(), NOW() + INTERVAL '30 days')
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    description = EXCLUDED.description,
    state = EXCLUDED.state, 
    status = EXCLUDED.status,
    scoring_policy = EXCLUDED.scoring_policy,
    active = EXCLUDED.active;

-- Data: agents.sql
-- Seed: agents
-- Demo agents for pilot_alpha and pilot_beta.

INSERT INTO agents (id, owner_user_id, game_id, name, description, active) VALUES
('agent-star-hunter', 'usr-pilot-001', 'starfighter', 'StarHunter', 'Aggressive hunter bot targeting opponent starfighter', TRUE),
('agent-star-evasive', 'usr-pilot-002', 'starfighter', 'StarEvasive', 'Defensive evasive bot focused on survival maneuvers', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    description = EXCLUDED.description, 
    active = EXCLUDED.active;

-- Data: submissions.sql
-- Seed: submissions
-- Baseline submissions linked to demo agent files.

INSERT INTO submissions (id, agent_id, version, code_path, language, status, active) VALUES
('sub-star-hunter-1', 'agent-star-hunter', 1, 'games/starfighter/examples/bot_random.py', 'python', 'ready', TRUE),
('sub-star-evasive-1', 'agent-star-evasive', 1, 'games/starfighter/examples/bot_evasive.py', 'python', 'ready', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    code_path = EXCLUDED.code_path,
    language = EXCLUDED.language,
    status = EXCLUDED.status, 
    active = EXCLUDED.active;

-- Data: contest_entries.sql
-- Seed: contest_entries
-- Enrolls demo bots into the initial Starfighter tournament, locking specific submissions.

INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status, enrolled_at) VALUES
('entry-hunter-001', 'starfighter-cup-2026', 'agent-star-hunter', 'usr-pilot-001', 'sub-star-hunter-1', 'enrolled', NOW()),
('entry-evasive-001', 'starfighter-cup-2026', 'agent-star-evasive', 'usr-pilot-002', 'sub-star-evasive-1', 'enrolled', NOW())
ON CONFLICT (contest_id, agent_id) DO UPDATE SET 
    submission_id = EXCLUDED.submission_id, 
    status = EXCLUDED.status;

-- Data: matches.sql
-- Seed: matches
-- Match instances created for tournaments or friendly matches.
-- Note: Matches are dynamically scheduled by the orchestrator, but an initial
-- completed demo match is provided for development and analytics preview.

INSERT INTO matches (id, contest_id, game_id, game_version, status, seed, active, created_at, finished_at) VALUES
('match-demo-001', 'starfighter-cup-2026', 'starfighter', '0.1.0', 'completed', 1337, TRUE, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '50 minutes')
ON CONFLICT (id) DO UPDATE SET 
    status = EXCLUDED.status, 
    finished_at = EXCLUDED.finished_at;

-- Data: rankings.sql
-- Seed: rankings
-- Initial leaderboard state for tournament entrants.

INSERT INTO rankings (id, contest_id, agent_id, user_id, score, points, matches_played, wins, losses, draws, disqualifications, tiebreaker_score, "rank", updated_at) VALUES
('rank-hunter-001', 'starfighter-cup-2026', 'agent-star-hunter', 'usr-pilot-001', 0, 0, 0, 0, 0, 0, 0, 0, 1, NOW()),
('rank-evasive-001', 'starfighter-cup-2026', 'agent-star-evasive', 'usr-pilot-002', 0, 0, 0, 0, 0, 0, 0, 0, 1, NOW())
ON CONFLICT (contest_id, agent_id) DO UPDATE SET 
    updated_at = EXCLUDED.updated_at;

-- Data: results.sql
-- Seed: results
-- Per-agent outcomes and scoring for completed matches.

INSERT INTO results (id, match_id, submission_id, score, "rank", status, details, created_at) VALUES
('res-demo-001-hunter', 'match-demo-001', 'sub-star-hunter-1', 100, 1, 'finished', 'Target destroyed at tick 124', NOW() - INTERVAL '50 minutes'),
('res-demo-001-evasive', 'match-demo-001', 'sub-star-evasive-1', 30, 2, 'finished', 'Hull integrity depleted', NOW() - INTERVAL '50 minutes')
ON CONFLICT (id) DO UPDATE SET 
    score = EXCLUDED.score, 
    "rank" = EXCLUDED."rank",
    status = EXCLUDED.status;

-- Data: replays.sql
-- Seed: replays
-- Execution replay metadata referencing artifacts stored on disk/S3.

INSERT INTO replays (id, match_id, file_path, duration_ticks, summary, sha256, size_bytes, active, created_at) VALUES
('rep-demo-001', 'match-demo-001', 'artifacts/replays/match-demo-001.ndjson.zst', 124, '1v1 Starfighter battle: StarHunter vs StarEvasive', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', 4096, TRUE, NOW() - INTERVAL '50 minutes')
ON CONFLICT (id) DO UPDATE SET 
    file_path = EXCLUDED.file_path, 
    duration_ticks = EXCLUDED.duration_ticks,
    size_bytes = EXCLUDED.size_bytes;
