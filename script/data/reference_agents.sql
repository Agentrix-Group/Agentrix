-- Participants & Reference Agents seed

-- Admin and Test participants (password: admin123)
INSERT INTO participants (id, username, email, password, role_id, active) VALUES
('part-admin', 'admin', 'admin@agentrix.ai', '7fcf4ba391c48784edde599889d6e3f1e47a27db36ecc050cc92f259bfac38afad2c68a1ae804d77075e8fb722503f3eca2b2c1006ee6f6c7b7628cb45fffd1d', 'admin', TRUE),
('part-bot-master', 'botmaster', 'master@agentrix.ai', '7fcf4ba391c48784edde599889d6e3f1e47a27db36ecc050cc92f259bfac38afad2c68a1ae804d77075e8fb722503f3eca2b2c1006ee6f6c7b7628cb45fffd1d', 'participant', TRUE)
ON DUPLICATE KEY UPDATE email = VALUES(email);

-- Reference Agents
INSERT INTO agents (id, participant_id, game_id, name, description, active) VALUES
('agent-evasive', 'part-admin', 'starfighter', 'Evasive Pilot', 'Reference pilot prioritizing movement and survival.', TRUE),
('agent-hunter', 'part-admin', 'starfighter', 'Hunter Pilot', 'Reference pilot pursuing and firing at its opponent.', TRUE)
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- Submissions for Reference Agents
INSERT INTO submissions (id, agent_id, version, code_path, language, status, active) VALUES
('sub-evasive-v1', 'agent-evasive', 1, 'games/starfighter/examples/bot_evasive.py', 'python', 'ready', TRUE),
('sub-hunter-v1', 'agent-hunter', 1, 'games/starfighter/examples/bot_hunter.py', 'python', 'ready', TRUE)
ON DUPLICATE KEY UPDATE status = VALUES(status);
