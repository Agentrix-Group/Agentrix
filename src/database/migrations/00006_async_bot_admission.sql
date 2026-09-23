-- +goose Up
-- +goose StatementBegin

-- ADR-0014 (N4) / ADR-0007: the admission dry run of a bot runs on the
-- worker, never on the API. A submission waits in 'validating' until a
-- worker claims it; the rejection reason is kept for its author.
ALTER TABLE submissions
ADD COLUMN IF NOT EXISTS error_detail TEXT,
ADD COLUMN IF NOT EXISTS admission_claimed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_submissions_validating
ON submissions (created_at)
WHERE status = 'validating';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_submissions_validating;
ALTER TABLE submissions DROP COLUMN IF EXISTS admission_claimed_at;
ALTER TABLE submissions DROP COLUMN IF EXISTS error_detail;

-- +goose StatementEnd
