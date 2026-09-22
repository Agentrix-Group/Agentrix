-- Seed: replays
-- Execution replay metadata referencing artifacts stored on disk/S3.

INSERT INTO replays (id, match_id, file_path, duration_ticks, summary, sha256, size_bytes, active, created_at) VALUES
('rep-demo-001', 'match-demo-001', 'artifacts/replays/match-demo-001.ndjson.zst', 124, '1v1 Starfighter battle: StarHunter vs StarEvasive', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', 4096, TRUE, NOW() - INTERVAL '50 minutes')
ON CONFLICT (id) DO UPDATE SET 
    file_path = EXCLUDED.file_path, 
    duration_ticks = EXCLUDED.duration_ticks,
    size_bytes = EXCLUDED.size_bytes;
