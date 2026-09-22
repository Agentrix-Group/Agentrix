-- Seed: rankings
-- Initial leaderboard state for tournament entrants.

INSERT INTO rankings (id, contest_id, agent_id, user_id, score, points, matches_played, wins, losses, draws, disqualifications, tiebreaker_score, "rank", updated_at) VALUES
('rank-hunter-001', 'starfighter-cup-2026', 'agent-star-hunter', 'usr-pilot-001', 0, 0, 0, 0, 0, 0, 0, 0, 1, NOW()),
('rank-evasive-001', 'starfighter-cup-2026', 'agent-star-evasive', 'usr-pilot-002', 0, 0, 0, 0, 0, 0, 0, 0, 1, NOW())
ON CONFLICT (contest_id, agent_id) DO UPDATE SET 
    updated_at = EXCLUDED.updated_at;
