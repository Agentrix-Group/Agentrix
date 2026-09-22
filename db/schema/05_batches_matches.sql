SET search_path = agentrix, pg_catalog;

CREATE TABLE evaluation_batches (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    game_release_id uuid NOT NULL,
    evaluation_policy_version_id uuid NOT NULL,
    scoring_policy_version_id uuid NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('practice', 'official', 'rejudge')),
    reference_kind text NOT NULL CHECK (reference_kind IN ('fixed_suite', 'sealed_roster', 'normalized_reference')),
    state text NOT NULL CHECK (state IN ('draft', 'sealed', 'running', 'completed', 'cancelled')),
    roster_digest sha256_digest,
    created_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    sealed_at timestamptz,
    completed_at timestamptz,
    UNIQUE (id, contest_id, contest_task_id),
    UNIQUE (id, test_case_id),
    UNIQUE (id, game_release_id),
    FOREIGN KEY (contest_id, contest_task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (game_release_id)
        REFERENCES game_releases(id) ON DELETE RESTRICT,
    FOREIGN KEY (evaluation_policy_version_id)
        REFERENCES evaluation_policy_versions(id) ON DELETE RESTRICT,
    FOREIGN KEY (scoring_policy_version_id)
        REFERENCES scoring_policy_versions(id) ON DELETE RESTRICT,
    CHECK ((state = 'draft') = (sealed_at IS NULL)),
    CHECK (state = 'draft' OR roster_digest IS NOT NULL),
    CHECK (completed_at IS NULL OR (sealed_at IS NOT NULL AND completed_at >= sealed_at))
);
CREATE INDEX evaluation_batches_task_state
    ON evaluation_batches(contest_task_id, purpose, state, created_at);

CREATE TABLE batch_roster (
    id uuid PRIMARY KEY,
    batch_id uuid NOT NULL REFERENCES evaluation_batches(id) ON DELETE RESTRICT,
    contest_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    source_kind text NOT NULL CHECK (source_kind IN ('submission', 'baseline')),
    contest_entry_id uuid,
    submission_id uuid,
    build_attempt_id uuid,
    baseline_program_id uuid,
    executable_artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    executable_digest sha256_digest NOT NULL,
    added_at timestamptz NOT NULL,
    UNIQUE (batch_id, id),
    FOREIGN KEY (batch_id, contest_id, contest_task_id)
        REFERENCES evaluation_batches(id, contest_id, contest_task_id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_entry_id, contest_task_id, submission_id)
        REFERENCES submissions(contest_id, contest_entry_id, contest_task_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (submission_id, build_attempt_id)
        REFERENCES build_attempts(submission_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (baseline_program_id)
        REFERENCES baseline_programs(id) ON DELETE RESTRICT,
    CHECK (
        (source_kind = 'submission' AND contest_entry_id IS NOT NULL AND submission_id IS NOT NULL
            AND build_attempt_id IS NOT NULL AND baseline_program_id IS NULL)
        OR
        (source_kind = 'baseline' AND contest_entry_id IS NULL AND submission_id IS NULL
            AND build_attempt_id IS NULL AND baseline_program_id IS NOT NULL)
    )
);
CREATE UNIQUE INDEX batch_roster_one_submission_per_entry
    ON batch_roster(batch_id, contest_entry_id) WHERE source_kind = 'submission';
CREATE UNIQUE INDEX batch_roster_submission_once
    ON batch_roster(batch_id, submission_id) WHERE source_kind = 'submission';
CREATE UNIQUE INDEX batch_roster_baseline_once
    ON batch_roster(batch_id, baseline_program_id) WHERE source_kind = 'baseline';

CREATE TABLE matches (
    id uuid PRIMARY KEY,
    batch_id uuid NOT NULL,
    contest_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    game_release_id uuid NOT NULL,
    test_case_id uuid,
    purpose text NOT NULL CHECK (purpose IN ('practice', 'official', 'rejudge', 'validation')),
    seed bigint NOT NULL,
    scenario_artifact_id uuid REFERENCES artifacts(id) ON DELETE RESTRICT,
    schedule_key text NOT NULL,
    specification_digest sha256_digest NOT NULL,
    allow_duplicate_programs boolean NOT NULL,
    state text NOT NULL CHECK (state IN ('planned', 'sealed', 'queued', 'running', 'completed', 'failed', 'cancelled')),
    accepted_attempt_id uuid,
    created_at timestamptz NOT NULL,
    sealed_at timestamptz,
    completed_at timestamptz,
    UNIQUE (batch_id, schedule_key),
    UNIQUE (id, batch_id),
    UNIQUE (id, contest_id, contest_task_id),
    FOREIGN KEY (batch_id, contest_id, contest_task_id)
        REFERENCES evaluation_batches(id, contest_id, contest_task_id) ON DELETE RESTRICT,
    FOREIGN KEY (batch_id, game_release_id)
        REFERENCES evaluation_batches(id, game_release_id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_task_id, test_case_id)
        REFERENCES test_cases(task_id, id) ON DELETE RESTRICT,
    CHECK ((state = 'planned') = (sealed_at IS NULL)),
    CHECK (completed_at IS NULL OR (sealed_at IS NOT NULL AND completed_at >= sealed_at)),
    CHECK ((accepted_attempt_id IS NOT NULL) = (state = 'completed'))
);
CREATE INDEX matches_batch_state ON matches(batch_id, state, created_at);

CREATE TABLE match_seats (
    id uuid PRIMARY KEY,
    match_id uuid NOT NULL,
    batch_id uuid NOT NULL,
    roster_item_id uuid NOT NULL,
    seat_index smallint NOT NULL CHECK (seat_index >= 0),
    created_at timestamptz NOT NULL,
    UNIQUE (match_id, seat_index),
    UNIQUE (match_id, id),
    UNIQUE (match_id, id, batch_id),
    FOREIGN KEY (match_id, batch_id)
        REFERENCES matches(id, batch_id) ON DELETE RESTRICT,
    FOREIGN KEY (batch_id, roster_item_id)
        REFERENCES batch_roster(batch_id, id) ON DELETE RESTRICT
);

CREATE TABLE match_jobs (
    id uuid PRIMARY KEY,
    match_id uuid NOT NULL UNIQUE REFERENCES matches(id) ON DELETE RESTRICT,
    state text NOT NULL CHECK (state IN ('available', 'leased', 'succeeded', 'dead', 'cancelled')),
    priority integer NOT NULL DEFAULT 0,
    available_at timestamptz NOT NULL,
    max_attempts smallint NOT NULL CHECK (max_attempts BETWEEN 1 AND 20),
    attempt_count smallint NOT NULL DEFAULT 0 CHECK (attempt_count >= 0 AND attempt_count <= max_attempts),
    lease_owner text,
    lease_expires_at timestamptz,
    fencing_token bigint NOT NULL DEFAULT 0 CHECK (fencing_token >= 0),
    last_heartbeat_at timestamptz,
    last_error_kind text CHECK (last_error_kind IN ('worker_infrastructure', 'bot', 'protocol', 'game', 'cancelled')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (id, match_id),
    CHECK (
        (state = 'leased' AND lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR (state <> 'leased' AND lease_owner IS NULL AND lease_expires_at IS NULL)
    )
);
CREATE INDEX match_jobs_claim
    ON match_jobs(priority DESC, available_at, id) WHERE state = 'available';
CREATE INDEX match_jobs_expired_lease
    ON match_jobs(lease_expires_at, id) WHERE state = 'leased';

CREATE TABLE match_attempts (
    id uuid PRIMARY KEY,
    match_id uuid NOT NULL REFERENCES matches(id) ON DELETE RESTRICT,
    job_id uuid NOT NULL,
    attempt_number smallint NOT NULL CHECK (attempt_number > 0),
    worker_id text NOT NULL,
    fencing_token bigint NOT NULL CHECK (fencing_token > 0),
    state text NOT NULL CHECK (state IN ('running', 'succeeded', 'failed', 'rejected')),
    failure_origin text CHECK (failure_origin IN ('worker_infrastructure', 'bot', 'protocol', 'game', 'cancelled')),
    failure_code text,
    scheduled_specification_digest sha256_digest NOT NULL,
    observed_engine_digest sha256_digest,
    observed_rules_digest sha256_digest,
    observed_protocol_version text,
    started_at timestamptz NOT NULL,
    heartbeat_at timestamptz NOT NULL,
    ended_at timestamptz,
    log_artifact_id uuid REFERENCES artifacts(id) ON DELETE RESTRICT,
    output_digest sha256_digest,
    UNIQUE (match_id, attempt_number),
    UNIQUE (match_id, id),
    UNIQUE (job_id, fencing_token),
    FOREIGN KEY (job_id, match_id)
        REFERENCES match_jobs(id, match_id) ON DELETE RESTRICT,
    CHECK (heartbeat_at >= started_at),
    CHECK (ended_at IS NULL OR ended_at >= started_at),
    CHECK ((state IN ('failed', 'rejected')) = (failure_origin IS NOT NULL)),
    CHECK ((state IN ('succeeded', 'failed', 'rejected')) = (ended_at IS NOT NULL))
);
CREATE INDEX match_attempts_match_state ON match_attempts(match_id, state, attempt_number);

ALTER TABLE matches
    ADD CONSTRAINT matches_accepted_attempt_same_match
    FOREIGN KEY (id, accepted_attempt_id)
    REFERENCES match_attempts(match_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE match_seat_results (
    match_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    seat_id uuid NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('completed', 'bot_error', 'disqualified', 'no_result')),
    engine_score numeric(24, 8),
    placement smallint CHECK (placement IS NULL OR placement > 0),
    metrics_artifact_id uuid REFERENCES artifacts(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (attempt_id, seat_id),
    FOREIGN KEY (match_id, attempt_id)
        REFERENCES match_attempts(match_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (match_id, seat_id)
        REFERENCES match_seats(match_id, id) ON DELETE RESTRICT,
    CHECK ((outcome = 'completed') = (engine_score IS NOT NULL AND placement IS NOT NULL))
);

CREATE TABLE match_replays (
    match_id uuid NOT NULL,
    attempt_id uuid NOT NULL,
    artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    format text NOT NULL,
    format_version text NOT NULL,
    duration_ticks bigint NOT NULL CHECK (duration_ticks >= 0),
    state text NOT NULL CHECK (state IN ('pending', 'verified', 'published', 'rejected')),
    verified_at timestamptz,
    published_at timestamptz,
    PRIMARY KEY (attempt_id),
    UNIQUE (artifact_id),
    FOREIGN KEY (match_id, attempt_id)
        REFERENCES match_attempts(match_id, id) ON DELETE RESTRICT,
    CHECK ((state IN ('verified', 'published')) = (verified_at IS NOT NULL)),
    CHECK ((state = 'published') = (published_at IS NOT NULL)),
    CHECK (published_at IS NULL OR published_at >= verified_at)
);
