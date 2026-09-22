-- PostgreSQL Data Seeds for Agentrix platform
-- Idempotent inserts using ON CONFLICT DO UPDATE / DO NOTHING.

-- Roles
INSERT INTO roles (id, description, active) VALUES
('admin', 'System Administrator', TRUE),
('organizer', 'Contest and tournament organizer', TRUE),
('player', 'Competitor and bot author', TRUE),
('participant', 'Competitor and bot author (legacy alias)', TRUE),
('referee', 'Tournament official and referee', TRUE),
('spectator', 'Public viewer and audience', TRUE)
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, active = EXCLUDED.active;

-- Permissions
INSERT INTO permissions (id, description, active) VALUES
('read', 'Read system data and resources', TRUE),
('write', 'Modify and update resources', TRUE),
('admin', 'System administration capabilities', TRUE),
('execute-match', 'Run and schedule matches', TRUE),
('submit-agent', 'Upload and manage bot submissions', TRUE),
('create-contest', 'Create and configure tournaments', TRUE),
('manage-users', 'Manage user accounts and role assignments', TRUE)
ON CONFLICT (id) DO UPDATE SET description = EXCLUDED.description, active = EXCLUDED.active;

-- Role Permissions mapping
INSERT INTO role_permissions (role_id, permission_id, active) VALUES
-- Admin has all permissions
('admin', 'read', TRUE),
('admin', 'write', TRUE),
('admin', 'admin', TRUE),
('admin', 'execute-match', TRUE),
('admin', 'submit-agent', TRUE),
('admin', 'create-contest', TRUE),
('admin', 'manage-users', TRUE),
-- Organizer can manage contests, submit agents, and run matches
('organizer', 'read', TRUE),
('organizer', 'write', TRUE),
('organizer', 'execute-match', TRUE),
('organizer', 'submit-agent', TRUE),
('organizer', 'create-contest', TRUE),
-- Player can read, write own resources, submit agents, and execute matches
('player', 'read', TRUE),
('player', 'write', TRUE),
('player', 'submit-agent', TRUE),
('player', 'execute-match', TRUE),
-- Participant (legacy alias of player)
('participant', 'read', TRUE),
('participant', 'write', TRUE),
('participant', 'submit-agent', TRUE),
('participant', 'execute-match', TRUE),
-- Referee can read and execute matches
('referee', 'read', TRUE),
('referee', 'execute-match', TRUE),
-- Spectator can read public resources
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
INSERT INTO contests (id, name, description, game_id, category_id, state, status, active, starts_at, ends_at) VALUES
('starfighter-cup-2026', 'Starfighter Championship 2026', 'Tournament for Python bots controlling starfighters.', 'starfighter', 'battle-royale', 'registration_open', 'upcoming', TRUE, NOW(), NOW() + INTERVAL '30 days')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, state = EXCLUDED.state, active = EXCLUDED.active;

-- Seed default users (with SHA-512 hashed passwords)
-- admin: admin123 (7fcf4ba391c48784edde599889d6e3f1e47a27db36ecc050cc92f259bfac38afad2c68a1ae804d77075e8fb722503f3eca2b2c1006ee6f6c7b7628cb45fffd1d)
-- pilot_alpha: pilot123 (5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27)
-- pilot_beta: pilot123 (5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27)
INSERT INTO users (id, username, email, password, role_id, active) VALUES
('usr-admin-001', 'admin', 'admin@agentrix.local', '7fcf4ba391c48784edde599889d6e3f1e47a27db36ecc050cc92f259bfac38afad2c68a1ae804d77075e8fb722503f3eca2b2c1006ee6f6c7b7628cb45fffd1d', 'admin', TRUE),
('usr-pilot-001', 'pilot_alpha', 'pilot_alpha@agentrix.local', '5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27', 'player', TRUE),
('usr-pilot-002', 'pilot_beta', 'pilot_beta@agentrix.local', '5047300a58f3abe4458de0b0eb4316a1dcad86db8683ebcfb3ccbd096a28dd71c1324759698b3126c16e4b993fc5d84de58ecd937edc37c6bb8ab0cfefb57e27', 'player', TRUE)
ON CONFLICT (username) DO UPDATE SET role_id = EXCLUDED.role_id, password = EXCLUDED.password, active = EXCLUDED.active;

-- Seed User Roles (RBAC mapping)
INSERT INTO user_roles (user_id, role_id)
SELECT id, 'admin' FROM users WHERE username = 'admin'
UNION ALL
SELECT id, 'player' FROM users WHERE username = 'pilot_alpha'
UNION ALL
SELECT id, 'player' FROM users WHERE username = 'pilot_beta'
ON CONFLICT (user_id, role_id) DO NOTHING;

-- Seed Demo Agents
INSERT INTO agents (id, owner_user_id, game_id, name, description, active) VALUES
('agent-star-hunter', 'usr-pilot-001', 'starfighter', 'StarHunter', 'Aggressive hunter bot targeting opponent starfighter', TRUE),
('agent-star-evasive', 'usr-pilot-002', 'starfighter', 'StarEvasive', 'Defensive evasive bot focused on survival maneuvers', TRUE)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, active = EXCLUDED.active;

-- Seed Demo Submissions
INSERT INTO submissions (id, agent_id, version, code_path, language, status, active) VALUES
('sub-star-hunter-1', 'agent-star-hunter', 1, 'games/starfighter/examples/bot_random.py', 'python', 'ready', TRUE),
('sub-star-evasive-1', 'agent-star-evasive', 1, 'games/starfighter/examples/bot_evasive.py', 'python', 'ready', TRUE)
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, active = EXCLUDED.active;

-- Seed Contest Entries (locking specific submission_id)
INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status, enrolled_at) VALUES
('entry-hunter-001', 'starfighter-cup-2026', 'agent-star-hunter', 'usr-pilot-001', 'sub-star-hunter-1', 'enrolled', NOW()),
('entry-evasive-001', 'starfighter-cup-2026', 'agent-star-evasive', 'usr-pilot-002', 'sub-star-evasive-1', 'enrolled', NOW())
ON CONFLICT (contest_id, agent_id) DO UPDATE SET submission_id = EXCLUDED.submission_id, status = EXCLUDED.status;

-- Seed Initial Leaderboard Rankings
INSERT INTO rankings (id, contest_id, agent_id, user_id, score, matches_played, wins, losses, draws, "rank", updated_at) VALUES
('rank-hunter-001', 'starfighter-cup-2026', 'agent-star-hunter', 'usr-pilot-001', 0, 0, 0, 0, 0, 1, NOW()),
('rank-evasive-001', 'starfighter-cup-2026', 'agent-star-evasive', 'usr-pilot-002', 0, 0, 0, 0, 0, 1, NOW())
ON CONFLICT (contest_id, agent_id) DO UPDATE SET updated_at = EXCLUDED.updated_at;
