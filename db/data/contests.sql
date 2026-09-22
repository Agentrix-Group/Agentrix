-- Seed: contests
-- Initial tournament: Starfighter Championship 2026.

INSERT INTO contests (id, name, description, game_id, category_id, state, status, scoring_policy, active, starts_at, ends_at) VALUES
('starfighter-cup-2026', 'Starfighter Championship 2026', 'Tournament for Python bots controlling starfighters.', 'starfighter', 'battle-royale', 'registration_open', 'upcoming', '{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}'::jsonb, TRUE, NOW(), NOW() + INTERVAL '30 days')
ON CONFLICT (id) DO UPDATE SET 
    name = EXCLUDED.name, 
    description = EXCLUDED.description,
    state = EXCLUDED.state, 
    status = EXCLUDED.status,
    scoring_policy = EXCLUDED.scoring_policy,
    active = EXCLUDED.active;
