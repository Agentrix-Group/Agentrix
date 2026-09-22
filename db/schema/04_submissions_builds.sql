SET search_path = agentrix, pg_catalog;

CREATE TABLE submissions (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL,
    contest_entry_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    submitted_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    source_artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    toolchain_id uuid NOT NULL REFERENCES toolchains(id) ON DELETE RESTRICT,
    submitted_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    idempotency_key text NOT NULL,
    disposition text NOT NULL CHECK (disposition IN ('eligible', 'ignored', 'withdrawn', 'disqualified')),
    disposition_reason text,
    CHECK (received_at >= submitted_at),
    CHECK ((disposition = 'eligible') = (disposition_reason IS NULL)),
    UNIQUE (contest_entry_id, contest_task_id, idempotency_key),
    UNIQUE (contest_id, contest_task_id, id),
    UNIQUE (contest_id, contest_entry_id, contest_task_id, id),
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_task_id, toolchain_id)
        REFERENCES task_toolchains(task_id, toolchain_id) ON DELETE RESTRICT
);
CREATE INDEX submissions_score_candidates
    ON submissions(contest_id, contest_entry_id, contest_task_id, submitted_at)
    WHERE disposition = 'eligible';
CREATE TRIGGER submissions_immutable_core
    BEFORE UPDATE OF contest_id, contest_entry_id, contest_task_id, submitted_by_user_id,
        source_artifact_id, toolchain_id, submitted_at, received_at, idempotency_key
    ON submissions FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();

CREATE TABLE submission_disposition_events (
    id uuid PRIMARY KEY,
    submission_id uuid NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
    previous_disposition text NOT NULL CHECK (previous_disposition IN ('eligible', 'ignored', 'withdrawn', 'disqualified')),
    new_disposition text NOT NULL CHECK (new_disposition IN ('eligible', 'ignored', 'withdrawn', 'disqualified')),
    reason text NOT NULL,
    changed_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    changed_at timestamptz NOT NULL,
    CHECK (previous_disposition <> new_disposition)
);

CREATE TABLE build_attempts (
    id uuid PRIMARY KEY,
    submission_id uuid NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
    attempt_number integer NOT NULL CHECK (attempt_number > 0),
    toolchain_id uuid NOT NULL REFERENCES toolchains(id) ON DELETE RESTRICT,
    state text NOT NULL CHECK (state IN ('pending', 'running', 'succeeded', 'failed', 'cancelled')),
    failure_kind text CHECK (failure_kind IN ('source', 'toolchain', 'infrastructure', 'protocol')),
    executable_artifact_id uuid REFERENCES artifacts(id) ON DELETE RESTRICT,
    log_artifact_id uuid REFERENCES artifacts(id) ON DELETE RESTRICT,
    started_at timestamptz,
    ended_at timestamptz,
    created_at timestamptz NOT NULL,
    UNIQUE (submission_id, attempt_number),
    UNIQUE (submission_id, id),
    CHECK (ended_at IS NULL OR (started_at IS NOT NULL AND ended_at >= started_at)),
    CHECK (
        (state = 'succeeded' AND executable_artifact_id IS NOT NULL AND failure_kind IS NULL AND ended_at IS NOT NULL)
        OR (state = 'failed' AND executable_artifact_id IS NULL AND failure_kind IS NOT NULL AND ended_at IS NOT NULL)
        OR (state IN ('pending', 'running', 'cancelled') AND executable_artifact_id IS NULL)
    )
);
CREATE INDEX build_attempts_submission_state ON build_attempts(submission_id, state, attempt_number);

