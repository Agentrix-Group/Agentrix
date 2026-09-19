BEGIN;

CREATE TABLE IF NOT EXISTS match_runs (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    worker_id VARCHAR(128) NOT NULL,
    fencing_token BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'committed', 'aborted', 'superseded')),
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    heartbeat_at TIMESTAMPTZ,
    last_error TEXT
);

-- Partial unique index: only one run per match can ever be in 'committed' state
CREATE UNIQUE INDEX IF NOT EXISTS idx_match_runs_committed
    ON match_runs(match_id)
    WHERE status = 'committed';

CREATE INDEX IF NOT EXISTS idx_match_runs_match_token
    ON match_runs(match_id, fencing_token);

-- Replay integrity columns
ALTER TABLE replays ADD COLUMN IF NOT EXISTS sha256 VARCHAR(64);
ALTER TABLE replays ADD COLUMN IF NOT EXISTS size_bytes BIGINT DEFAULT 0;

COMMIT;
