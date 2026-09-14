-- Categories & Games seed
INSERT INTO categories (id, description, active) VALUES
('ai-challenge', 'Artificial Intelligence & Algorithmic Bots', TRUE),
('battle-royale', 'Multi-Agent Survival Arena', TRUE),
('heuristics', 'Heuristic Strategy & Optimization', TRUE)
ON DUPLICATE KEY UPDATE description = VALUES(description);

-- Games seed
INSERT INTO games (id, name, description, manifest_path, min_players, max_players, active) VALUES
('arena-basica', 'Arena Basica', 'Canonical 2-4 agent grid survival battle with health, energy, attacks and shields.', 'games/arena-basica/manifest.yaml', 2, 4, TRUE)
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description);

-- Initial Contest seed
INSERT INTO contests (id, name, description, game_id, category_id, status, active) VALUES
('arena-cup-2026', 'Arena Basica Spring Championship 2026', 'Inaugural tournament for autonomous agent bots in Arena Basica.', 'arena-basica', 'battle-royale', 'active', TRUE)
ON DUPLICATE KEY UPDATE name = VALUES(name);
