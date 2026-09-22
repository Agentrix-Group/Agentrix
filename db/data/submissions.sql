-- Seed: submissions
-- Baseline submissions linked to demo agent files.

INSERT INTO submissions (id, agent_id, version, code_path, language, status, active) VALUES
('sub-star-hunter-1', 'agent-star-hunter', 1, 'games/starfighter/examples/bot_random.py', 'python', 'ready', TRUE),
('sub-star-evasive-1', 'agent-star-evasive', 1, 'games/starfighter/examples/bot_evasive.py', 'python', 'ready', TRUE)
ON CONFLICT (id) DO UPDATE SET 
    code_path = EXCLUDED.code_path,
    language = EXCLUDED.language,
    status = EXCLUDED.status, 
    active = EXCLUDED.active;
