-- Seed: results
-- Per-agent outcomes and scoring for completed matches.

INSERT INTO results (id, match_id, submission_id, score, "rank", status, details, created_at) VALUES
('res-demo-001-hunter', 'match-demo-001', 'sub-star-hunter-1', 100, 1, 'finished', 'Target destroyed at tick 124', NOW() - INTERVAL '50 minutes'),
('res-demo-001-evasive', 'match-demo-001', 'sub-star-evasive-1', 30, 2, 'finished', 'Hull integrity depleted', NOW() - INTERVAL '50 minutes')
ON CONFLICT (id) DO UPDATE SET 
    score = EXCLUDED.score, 
    "rank" = EXCLUDED."rank",
    status = EXCLUDED.status;
