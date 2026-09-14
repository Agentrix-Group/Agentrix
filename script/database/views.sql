-- Views for Agentrix platform

CREATE OR REPLACE VIEW view_contest_rankings AS
SELECT 
    r.id AS ranking_id,
    r.contest_id,
    c.name AS contest_name,
    r.agent_id,
    a.name AS agent_name,
    r.participant_id,
    p.username AS participant_username,
    r.score,
    r.matches_played,
    r.wins,
    r.losses,
    r.draws,
    r.`rank`,
    r.updated_at
FROM rankings r
JOIN contests c ON r.contest_id = c.id
JOIN agents a ON r.agent_id = a.id
JOIN participants p ON r.participant_id = p.id;

CREATE OR REPLACE VIEW view_match_summaries AS
SELECT 
    m.id AS match_id,
    m.contest_id,
    c.name AS contest_name,
    m.game_id,
    g.name AS game_name,
    m.status,
    m.seed,
    m.replay_id,
    m.created_at,
    m.finished_at,
    COUNT(res.id) AS total_participants
FROM matches m
LEFT JOIN contests c ON m.contest_id = c.id
LEFT JOIN games g ON m.game_id = g.id
LEFT JOIN results res ON m.id = res.match_id
GROUP BY m.id, m.contest_id, c.name, m.game_id, g.name, m.status, m.seed, m.replay_id, m.created_at, m.finished_at;
