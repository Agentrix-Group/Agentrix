BEGIN;

CREATE TABLE IF NOT EXISTS match_jobs (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    contest_id VARCHAR(64) REFERENCES contests(id) ON DELETE SET NULL,
    game_id VARCHAR(64) NOT NULL DEFAULT 'starfighter',
    submission_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    seed BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'reserved', 'completed', 'failed')),
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reserved_at TIMESTAMPTZ,
    reserved_by VARCHAR(128),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_match_jobs_reservation
    ON match_jobs(status, available_at, created_at);

COMMIT;
