-- Roles and role-permission mappings seed
INSERT INTO roles (id, description, active) VALUES
('admin', 'System Administrator', TRUE),
('participant', 'Competitor and bot author', TRUE),
('referee', 'Tournament official and referee', TRUE),
('spectator', 'Public viewer and audience', TRUE)
ON DUPLICATE KEY UPDATE description = VALUES(description);

-- Role Permissions
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
-- Admin has all
('admin', 'read', TRUE),
('admin', 'write', TRUE),
('admin', 'admin', TRUE),
('admin', 'execute-match', TRUE),
('admin', 'submit-agent', TRUE),

-- Participant can read, submit agents, and run matches
('participant', 'read', TRUE),
('participant', 'write', TRUE),
('participant', 'submit-agent', TRUE),
('participant', 'execute-match', TRUE),

-- Referee can read and execute matches
('referee', 'read', TRUE),
('referee', 'execute-match', TRUE),

-- Spectator can read
('spectator', 'read', TRUE)
ON DUPLICATE KEY UPDATE active = VALUES(active);
