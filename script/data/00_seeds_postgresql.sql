-- PostgreSQL Data Seeds for Agentrix platform
-- Idempotent inserts using ON CONFLICT (id) DO UPDATE.

-- Roles
INSERT INTO roles (id, description, active) VALUES
('admin', 'System Administrator', TRUE),
('participant', 'Competitor and bot author', TRUE),
('referee', 'Tournament official and referee', TRUE),
('spectator', 'Public viewer and audience', TRUE)
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, active = EXCLUDED.active;

-- Permissions
INSERT INTO permissions (id, description, active) VALUES
('read', 'Read system data and resources', TRUE),
('write', 'Modify and update resources', TRUE),
('admin', 'System administration capabilities', TRUE),
('execute-match', 'Run and schedule matches', TRUE),
('submit-agent', 'Upload and manage bot submissions', TRUE)
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, active = EXCLUDED.active;

-- Role Permissions mapping
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
-- Admin has all
('admin', 'read', TRUE),
('admin', 'write', TRUE),
('admin', 'admin', TRUE),
('admin', 'execute-match', TRUE),
('admin', 'submit-agent', TRUE),
-- Participant can read, write, submit agents, and run matches
('participant', 'read', TRUE),
('participant', 'write', TRUE),
('participant', 'submit-agent', TRUE),
('participant', 'execute-match', TRUE),
-- Referee can read and execute matches
('referee', 'read', TRUE),
('referee', 'execute-match', TRUE),
-- Spectator can read
('spectator', 'read', TRUE)
ON CONFLICT (role_id, permission_id) DO UPDATE SET active = EXCLUDED.active;

-- Categories
INSERT INTO categories (id, description, active) VALUES
('ai-challenge', 'Artificial Intelligence & Algorithmic Bots', TRUE),
('battle-royale', 'Deterministic Starfighter Duel', TRUE),
('heuristics', 'Heuristic Strategy & Optimization', TRUE)
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, active = EXCLUDED.active;

-- Games
INSERT INTO games (id, name, description, manifest_path, min_players, max_players, active) VALUES
('starfighter', 'Starfighter Arena', 'Deterministic duel between two Python-controlled starfighters.', 'games/starfighter/manifest.yaml', 2, 2, TRUE)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, active = EXCLUDED.active;

-- Initial Demo Contest
INSERT INTO contests (id, name, description, game_id, category_id, state, status, active) VALUES
('starfighter-cup-2026', 'Starfighter Championship 2026', 'Tournament for Python bots controlling starfighters.', 'starfighter', 'battle-royale', 'registration_open', 'upcoming', TRUE)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, state = EXCLUDED.state, active = EXCLUDED.active;

-- Seed default admin user (password: admin123 hashed with SHA-512)
-- SHA512("admin123") = c7ad44cbad762a5da0a452f9e854fdc1e0e7a52a38015f23f3eab1d80b931dd472634dfac71cd34ebc35d16ab7fb8a90c81f975113d6c7538dc69dd8de9077ec
INSERT INTO participants (id, username, email, password, role_id, active) VALUES
('admin-root-uuid-001', 'admin', 'admin@agentrix.local', 'c7ad44cbad762a5da0a452f9e854fdc1e0e7a52a38015f23f3eab1d80b931dd472634dfac71cd34ebc35d16ab7fb8a90c81f975113d6c7538dc69dd8de9077ec', 'admin', TRUE)
ON CONFLICT (username) DO UPDATE SET role_id = EXCLUDED.role_id, password = EXCLUDED.password, active = EXCLUDED.active;
