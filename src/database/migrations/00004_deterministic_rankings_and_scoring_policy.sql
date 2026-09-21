-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- 1. Contest Scoring Policy
-- ============================================================================
ALTER TABLE contests 
ADD COLUMN IF NOT EXISTS scoring_policy JSONB NOT NULL 
DEFAULT '{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}'::jsonb;

-- ============================================================================
-- 2. Granular Ranking Accumulators & Tiebreakers
-- ============================================================================
ALTER TABLE rankings
ADD COLUMN IF NOT EXISTS points INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS disqualifications INT NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS tiebreaker_score DOUBLE PRECISION NOT NULL DEFAULT 0;

-- Backfill points from score for existing rankings
UPDATE rankings SET points = score WHERE points = 0 AND score != 0;

-- ============================================================================
-- 3. Idempotent Ranking Execution Tracking (ranking_applied_runs)
-- ============================================================================
CREATE TABLE IF NOT EXISTS ranking_applied_runs (
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    run_id VARCHAR(64) NOT NULL REFERENCES match_runs(id) ON DELETE CASCADE,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (contest_id, run_id)
);

CREATE INDEX IF NOT EXISTS idx_ranking_applied_runs_contest ON ranking_applied_runs(contest_id);
CREATE INDEX IF NOT EXISTS idx_ranking_applied_runs_match ON ranking_applied_runs(match_id);

-- ============================================================================
-- 4. Explicit Published Ranking Snapshots
-- ============================================================================
CREATE TABLE IF NOT EXISTS contest_rankings_snapshots (
    id VARCHAR(64) PRIMARY KEY,
    contest_id VARCHAR(64) NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    version INT NOT NULL,
    snapshot_data JSONB NOT NULL,
    published_by VARCHAR(64) REFERENCES users(id) ON DELETE SET NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_contest_rankings_version UNIQUE (contest_id, version)
);

CREATE INDEX IF NOT EXISTS idx_contest_rankings_snapshots_lookup 
ON contest_rankings_snapshots(contest_id, version DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS contest_rankings_snapshots CASCADE;
DROP TABLE IF EXISTS ranking_applied_runs CASCADE;
ALTER TABLE rankings DROP COLUMN IF EXISTS tiebreaker_score;
ALTER TABLE rankings DROP COLUMN IF EXISTS disqualifications;
ALTER TABLE rankings DROP COLUMN IF EXISTS points;
ALTER TABLE contests DROP COLUMN IF EXISTS scoring_policy;
-- +goose StatementEnd
