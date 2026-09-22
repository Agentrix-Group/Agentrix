-- Seed: categories
-- Initial contest categories.

INSERT INTO categories (id, description, active) VALUES
('ai-challenge', 'Artificial Intelligence & Algorithmic Bots', TRUE),
('battle-royale', 'Deterministic Starfighter Duel', TRUE),
('heuristics', 'Heuristic Strategy & Optimization', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    description = EXCLUDED.description, 
    active = EXCLUDED.active;
