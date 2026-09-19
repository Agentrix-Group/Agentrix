-- Categories & Games seed
INSERT INTO categories (id, description, active) VALUES
('ai-challenge', 'Artificial Intelligence & Algorithmic Bots', TRUE),
('battle-royale', 'Deterministic Starfighter Duel', TRUE),
('heuristics', 'Heuristic Strategy & Optimization', TRUE)
ON DUPLICATE KEY UPDATE description = VALUES(description);

-- Games seed
INSERT INTO games (id, name, description, manifest_path, min_players, max_players, active) VALUES
('starfighter', 'Starfighter Arena', 'Deterministic duel between two Python-controlled starfighters.', 'games/starfighter/manifest.yaml', 2, 2, TRUE)
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description);

-- Initial Contest seed
INSERT INTO contests (id, name, description, game_id, category_id, status, active) VALUES
('starfighter-cup-2026', 'Starfighter Championship 2026', 'Tournament for Python bots controlling starfighters.', 'starfighter', 'battle-royale', 'active', TRUE)
ON DUPLICATE KEY UPDATE name = VALUES(name);
