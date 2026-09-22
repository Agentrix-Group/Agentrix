-- Seed: roles
-- Defines canonical platform roles for RBAC.

INSERT INTO roles (id, description, active) VALUES
('admin', 'System Administrator', TRUE),
('organizer', 'Contest and tournament organizer', TRUE),
('player', 'Competitor and bot author', TRUE),
('participant', 'Competitor and bot author (legacy alias)', TRUE),
('referee', 'Tournament official and referee', TRUE),
('spectator', 'Public viewer and audience', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    description = EXCLUDED.description, 
    active = EXCLUDED.active;
