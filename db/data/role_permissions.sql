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
