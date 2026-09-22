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
