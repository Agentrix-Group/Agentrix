-- Seed: matches
-- Match instances created for tournaments or friendly matches.
-- Note: Matches are dynamically scheduled by the orchestrator, but an initial
-- completed demo match is provided for development and analytics preview.

INSERT INTO matches (id, contest_id, game_id, game_version, status, seed, active, created_at, finished_at) VALUES
('match-demo-001', 'starfighter-cup-2026', 'starfighter', '0.1.0', 'completed', 1337, TRUE, NOW() - INTERVAL '1 hour', NOW() - INTERVAL '50 minutes')
ON CONFLICT (id) DO UPDATE SET 
    status = EXCLUDED.status, 
    finished_at = EXCLUDED.finished_at;
