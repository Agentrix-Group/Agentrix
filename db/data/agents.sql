-- Seed: agents
-- Demo agents for pilot_alpha and pilot_beta.

INSERT INTO agents (id, owner_user_id, game_id, name, description, active) VALUES
('agent-star-hunter', 'usr-pilot-001', 'starfighter', 'StarHunter', 'Aggressive hunter bot targeting opponent starfighter', TRUE),
('agent-star-evasive', 'usr-pilot-002', 'starfighter', 'StarEvasive', 'Defensive evasive bot focused on survival maneuvers', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    description = EXCLUDED.description, 
    active = EXCLUDED.active;
