-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- 1. Sessions & Refresh Token Persistence
-- ============================================================================
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
-- 2. Canonical Dynamic RBAC Capabilities
-- ============================================================================
INSERT INTO permissions (id, description, active) VALUES
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
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, active = EXCLUDED.active;

-- Populate role_permissions for each role

-- 1. Admin: full access to all capabilities
INSERT INTO role_permissions (role_id, permission_id, active)
SELECT 'admin', p.id, TRUE
FROM permissions p
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- 2. Organizer: tournament management, match execution, view all submissions/agents
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
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

-- 3. Player / Participant: own agent/submission management, contest enrollment, match/ranking viewing
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
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
('player', 'replays:view', TRUE),
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

-- 4. Referee: match execution & review
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('referee', 'matches:view', TRUE),
('referee', 'matches:run', TRUE),
('referee', 'rankings:view', TRUE),
('referee', 'replays:view', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- 5. Spectator: public read only
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
('spectator', 'contests:view', TRUE),
('spectator', 'matches:view', TRUE),
('spectator', 'rankings:view', TRUE),
('spectator', 'replays:view', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sessions CASCADE;
-- +goose StatementEnd
