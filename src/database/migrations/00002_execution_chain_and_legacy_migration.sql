-- +goose Up
-- +goose StatementBegin

-- ============================================================================
-- 1. Legacy Schema Detection & Safe Data Migration (Participant -> User)
-- ============================================================================
DO $$
DECLARE
    has_participants BOOLEAN;
    has_name_col BOOLEAN;
    has_username_col BOOLEAN;
    has_email_col BOOLEAN;
    has_pass_col BOOLEAN;
    has_role_col BOOLEAN;
    has_active_col BOOLEAN;
    has_created_col BOOLEAN;
    username_expr TEXT := '''user_'' || SUBSTRING(p.id, 1, 8)';
    email_expr TEXT := '''user_'' || SUBSTRING(p.id, 1, 8) || ''@agentrix.local''';
    pass_expr TEXT := '''disabled_hash''';
    role_expr TEXT := '''player''';
    active_expr TEXT := 'TRUE';
    created_expr TEXT := 'CURRENT_TIMESTAMP';
    insert_sql TEXT;
BEGIN
    SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'participants') INTO has_participants;
    IF has_participants THEN
        SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'participants' AND column_name = 'username') INTO has_username_col;
        SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'participants' AND column_name = 'name') INTO has_name_col;
        SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'participants' AND column_name = 'email') INTO has_email_col;
        SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'participants' AND column_name = 'password') INTO has_pass_col;
        SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'participants' AND column_name = 'role_id') INTO has_role_col;
        SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'participants' AND column_name = 'active') INTO has_active_col;
        SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'participants' AND column_name = 'created_at') INTO has_created_col;

        IF has_username_col AND has_name_col THEN
            username_expr := 'COALESCE(NULLIF(p.username, ''''), NULLIF(p.name, ''''), ''user_'' || SUBSTRING(p.id, 1, 8))';
        ELSIF has_username_col THEN
            username_expr := 'COALESCE(NULLIF(p.username, ''''), ''user_'' || SUBSTRING(p.id, 1, 8))';
        ELSIF has_name_col THEN
            username_expr := 'COALESCE(NULLIF(p.name, ''''), ''user_'' || SUBSTRING(p.id, 1, 8))';
        END IF;

        IF has_email_col THEN
            email_expr := 'COALESCE(NULLIF(p.email, ''''), ''user_'' || SUBSTRING(p.id, 1, 8) || ''@agentrix.local'')';
        END IF;

        IF has_pass_col THEN
            pass_expr := 'COALESCE(NULLIF(p.password, ''''), ''disabled_hash'')';
        END IF;

        IF has_role_col THEN
            role_expr := 'COALESCE(NULLIF(p.role_id, ''''), ''player'')';
        END IF;

        IF has_active_col THEN
            active_expr := 'COALESCE(p.active, TRUE)';
        END IF;

        IF has_created_col THEN
            created_expr := 'COALESCE(p.created_at, CURRENT_TIMESTAMP)';
        END IF;

        insert_sql := 'INSERT INTO users (id, username, email, password, role_id, active, created_at) ' ||
                      'SELECT p.id, ' || username_expr || ', ' || email_expr || ', ' || pass_expr || ', ' || role_expr || ', ' || active_expr || ', ' || created_expr || ' ' ||
                      'FROM participants p ON CONFLICT (id) DO NOTHING';

        EXECUTE insert_sql;
    END IF;

    -- If agents table has participant_id column and not owner_user_id, rename it
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'agents' AND column_name = 'participant_id'
    ) AND NOT EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'agents' AND column_name = 'owner_user_id'
    ) THEN
        ALTER TABLE agents RENAME COLUMN participant_id TO owner_user_id;
    END IF;

    -- If rankings table has participant_id column and not user_id, rename it
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'rankings' AND column_name = 'participant_id'
    ) AND NOT EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'rankings' AND column_name = 'user_id'
    ) THEN
        ALTER TABLE rankings RENAME COLUMN participant_id TO user_id;
    END IF;

    -- If contest_entries table has participant_id column and not user_id, rename it
    IF EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'contest_entries' AND column_name = 'participant_id'
    ) AND NOT EXISTS (
        SELECT FROM information_schema.columns 
        WHERE table_schema = 'public' AND table_name = 'contest_entries' AND column_name = 'user_id'
    ) THEN
        ALTER TABLE contest_entries RENAME COLUMN participant_id TO user_id;
    END IF;
END $$;

-- ============================================================================
-- 2. RBAC Many-to-Many: user_roles
-- ============================================================================
CREATE TABLE IF NOT EXISTS user_roles (
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id VARCHAR(64) NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

-- Backfill user_roles from users.role_id
INSERT INTO user_roles (user_id, role_id)
SELECT id, role_id FROM users
WHERE role_id IS NOT NULL AND role_id != ''
ON CONFLICT (user_id, role_id) DO NOTHING;

-- ============================================================================
-- 3. ContestEntry Version Locking: submission_id
-- ============================================================================
ALTER TABLE contest_entries ADD COLUMN IF NOT EXISTS submission_id VARCHAR(64) REFERENCES submissions(id) ON DELETE RESTRICT;

-- Backfill submission_id for any existing contest_entries:
-- 1. Try to find the latest ready submission for the agent
UPDATE contest_entries ce
SET submission_id = (
    SELECT s.id FROM submissions s
    WHERE s.agent_id = ce.agent_id
    ORDER BY s.version DESC, s.created_at DESC
    LIMIT 1
)
WHERE ce.submission_id IS NULL;

-- 2. If an agent has no submissions yet, create a baseline submission
INSERT INTO submissions (id, agent_id, version, code_path, language, status, active)
SELECT 'sub-init-' || SUBSTRING(ce.agent_id, 1, 20) || '-' || SUBSTRING(ce.id, 1, 8),
       ce.agent_id, 1, 'games/starfighter/examples/bot_random.py', 'python', 'ready', TRUE
FROM contest_entries ce
WHERE ce.submission_id IS NULL
ON CONFLICT (id) DO NOTHING;

-- 3. Link the generated baseline submission
UPDATE contest_entries ce
SET submission_id = 'sub-init-' || SUBSTRING(ce.agent_id, 1, 20) || '-' || SUBSTRING(ce.id, 1, 8)
WHERE ce.submission_id IS NULL;

-- 4. Enforce NOT NULL and create index
ALTER TABLE contest_entries ALTER COLUMN submission_id SET NOT NULL;
CREATE INDEX IF NOT EXISTS idx_contest_entries_submission_id ON contest_entries(submission_id);

-- 5. Expand contest_entries status to include 'active'
ALTER TABLE contest_entries DROP CONSTRAINT IF EXISTS contest_entries_status_check;
ALTER TABLE contest_entries ADD CONSTRAINT contest_entries_status_check
    CHECK (status IN ('enrolled', 'active', 'disqualified', 'withdrawn'));

-- ============================================================================
-- 4. Match Slots: Ordered, Immutable Slots per Match
-- ============================================================================
CREATE TABLE IF NOT EXISTS match_slots (
    id VARCHAR(64) PRIMARY KEY,
    match_id VARCHAR(64) NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    slot_index INT NOT NULL,
    contest_entry_id VARCHAR(64) REFERENCES contest_entries(id) ON DELETE SET NULL,
    submission_id VARCHAR(64) NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
    agent_name VARCHAR(128) NOT NULL DEFAULT '',
    username VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_match_slots_index UNIQUE (match_id, slot_index)
);

CREATE INDEX IF NOT EXISTS idx_match_slots_match_id ON match_slots(match_id);
CREATE INDEX IF NOT EXISTS idx_match_slots_submission_id ON match_slots(submission_id);

-- ============================================================================
-- 5. MatchRuns Enhancements: 'created' Status & execution_spec
-- ============================================================================
ALTER TABLE match_runs DROP CONSTRAINT IF EXISTS match_runs_status_check;
ALTER TABLE match_runs ADD CONSTRAINT match_runs_status_check
    CHECK (status IN ('created', 'dispatching', 'running', 'completed', 'committed', 'failed', 'aborted', 'timed_out', 'superseded'));

ALTER TABLE match_runs ADD COLUMN IF NOT EXISTS execution_spec JSONB;
ALTER TABLE match_runs ADD COLUMN IF NOT EXISTS attempt INT NOT NULL DEFAULT 1;
ALTER TABLE match_runs ALTER COLUMN worker_id SET DEFAULT 'unassigned';
ALTER TABLE match_runs ALTER COLUMN fencing_token SET DEFAULT 0;

-- ============================================================================
-- 6. Results & Replays Association to Specific MatchRun & Slot
-- ============================================================================
ALTER TABLE results ADD COLUMN IF NOT EXISTS match_run_id VARCHAR(64) REFERENCES match_runs(id) ON DELETE RESTRICT;
ALTER TABLE results ADD COLUMN IF NOT EXISTS slot_id VARCHAR(64) REFERENCES match_slots(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_results_match_run_id ON results(match_run_id);
CREATE INDEX IF NOT EXISTS idx_results_slot_id ON results(slot_id);

ALTER TABLE replays ADD COLUMN IF NOT EXISTS match_run_id VARCHAR(64) REFERENCES match_runs(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS idx_replays_match_run_id ON replays(match_run_id);

-- ============================================================================
-- 7. Matches: committed_run_id Reference
-- ============================================================================
ALTER TABLE matches ADD COLUMN IF NOT EXISTS committed_run_id VARCHAR(64) REFERENCES match_runs(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_matches_committed_run_id ON matches(committed_run_id);

-- ============================================================================
-- 8. Historical Data Protection: Restrict Deletion of Historical Entities
-- ============================================================================
DO $$
BEGIN
    -- Protect matches from game cascade deletion
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'matches_game_id_fkey' AND table_name = 'matches'
    ) THEN
        ALTER TABLE matches DROP CONSTRAINT matches_game_id_fkey;
        ALTER TABLE matches ADD CONSTRAINT matches_game_id_fkey
            FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE RESTRICT;
    END IF;
END $$;

-- ============================================================================
-- 9. Idempotency Keys for Execution Requests
-- ============================================================================
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(255) PRIMARY KEY,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    response_body JSONB NOT NULL,
    status_code INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_resource ON idempotency_keys(resource_type, resource_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS idempotency_keys CASCADE;
ALTER TABLE matches DROP COLUMN IF EXISTS committed_run_id;
ALTER TABLE replays DROP COLUMN IF EXISTS match_run_id;
ALTER TABLE results DROP COLUMN IF EXISTS slot_id;
ALTER TABLE results DROP COLUMN IF EXISTS match_run_id;
ALTER TABLE match_runs DROP COLUMN IF EXISTS attempt;
ALTER TABLE match_runs DROP COLUMN IF EXISTS execution_spec;
DROP TABLE IF EXISTS match_slots CASCADE;
ALTER TABLE contest_entries DROP COLUMN IF EXISTS submission_id;
DROP TABLE IF EXISTS user_roles CASCADE;
-- +goose StatementEnd

