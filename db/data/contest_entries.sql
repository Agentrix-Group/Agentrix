-- Seed: contest_entries
-- Enrolls demo bots into the initial Starfighter tournament, locking specific submissions.

INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status, enrolled_at) VALUES
('entry-hunter-001', 'starfighter-cup-2026', 'agent-star-hunter', 'usr-pilot-001', 'sub-star-hunter-1', 'enrolled', NOW()),
('entry-evasive-001', 'starfighter-cup-2026', 'agent-star-evasive', 'usr-pilot-002', 'sub-star-evasive-1', 'enrolled', NOW())
ON CONFLICT (contest_id, agent_id) DO UPDATE SET 
    submission_id = EXCLUDED.submission_id, 
    status = EXCLUDED.status;
