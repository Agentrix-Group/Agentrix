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
