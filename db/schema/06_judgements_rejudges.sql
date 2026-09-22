SET search_path = agentrix, pg_catalog;

CREATE TABLE judgements (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    submission_id uuid NOT NULL,
    batch_id uuid NOT NULL,
    build_attempt_id uuid,
    evaluation_policy_version_id uuid NOT NULL REFERENCES evaluation_policy_versions(id) ON DELETE RESTRICT,
    scoring_policy_version_id uuid NOT NULL REFERENCES scoring_policy_versions(id) ON DELETE RESTRICT,
    score_scope text NOT NULL CHECK (score_scope IN ('practice', 'official')),
    state text NOT NULL CHECK (state IN ('pending', 'complete', 'awaiting_verification', 'verified', 'rejected')),
    verdict text NOT NULL CHECK (verdict IN ('accepted', 'wrong_answer', 'compile_error', 'protocol_error', 'bot_error', 'disqualified', 'infrastructure_error')),
    engine_score numeric(24, 8),
    task_points nonnegative_score,
    penalty_seconds bigint NOT NULL DEFAULT 0 CHECK (penalty_seconds >= 0),
    wins integer NOT NULL DEFAULT 0 CHECK (wins >= 0),
    draws integer NOT NULL DEFAULT 0 CHECK (draws >= 0),
    losses integer NOT NULL DEFAULT 0 CHECK (losses >= 0),
    effective boolean NOT NULL DEFAULT false,
    judged_at timestamptz NOT NULL,
    verified_by_user_id uuid REFERENCES users(id) ON DELETE RESTRICT,
    verified_at timestamptz,
    effective_at timestamptz,
    superseded_at timestamptz,
    UNIQUE (id, submission_id, batch_id),
    UNIQUE (id, submission_id),
    UNIQUE (id, contest_task_id),
    FOREIGN KEY (contest_id, contest_task_id, submission_id)
        REFERENCES submissions(contest_id, contest_task_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (batch_id, contest_id, contest_task_id)
        REFERENCES evaluation_batches(id, contest_id, contest_task_id) ON DELETE RESTRICT,
    FOREIGN KEY (submission_id, build_attempt_id)
        REFERENCES build_attempts(submission_id, id) ON DELETE RESTRICT,
    CHECK ((verified_by_user_id IS NULL) = (verified_at IS NULL)),
    CHECK (NOT effective OR (effective_at IS NOT NULL AND superseded_at IS NULL)),
    CHECK (superseded_at IS NULL OR (effective_at IS NOT NULL AND superseded_at >= effective_at)),
    CHECK (NOT effective OR state IN ('complete', 'verified')),
    CHECK (verdict = 'compile_error' OR build_attempt_id IS NOT NULL),
    CHECK (verdict <> 'infrastructure_error' OR task_points IS NULL)
);
CREATE UNIQUE INDEX judgements_one_effective_official
    ON judgements(submission_id, contest_task_id)
    WHERE effective AND score_scope = 'official';
CREATE INDEX judgements_scoring
    ON judgements(contest_id, contest_task_id, submission_id, effective, judged_at);

CREATE TABLE judgement_cases (
    judgement_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    test_case_id uuid NOT NULL,
    verdict text NOT NULL CHECK (verdict IN ('accepted', 'wrong_answer', 'protocol_error', 'bot_error', 'disqualified')),
    engine_score numeric(24, 8),
    task_points nonnegative_score,
    feedback_artifact_id uuid REFERENCES artifacts(id) ON DELETE RESTRICT,
    PRIMARY KEY (judgement_id, test_case_id),
    FOREIGN KEY (judgement_id, contest_task_id)
        REFERENCES judgements(id, contest_task_id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_task_id, test_case_id)
        REFERENCES test_cases(task_id, id) ON DELETE RESTRICT
);

CREATE TABLE judgement_matches (
    judgement_id uuid NOT NULL,
    submission_id uuid NOT NULL,
    batch_id uuid NOT NULL,
    match_id uuid NOT NULL,
    seat_id uuid NOT NULL,
    test_case_id uuid,
    PRIMARY KEY (judgement_id, match_id, seat_id),
    FOREIGN KEY (judgement_id, submission_id, batch_id)
        REFERENCES judgements(id, submission_id, batch_id) ON DELETE RESTRICT,
    FOREIGN KEY (match_id, batch_id)
        REFERENCES matches(id, batch_id) ON DELETE RESTRICT,
    FOREIGN KEY (match_id, seat_id, batch_id)
        REFERENCES match_seats(match_id, id, batch_id) ON DELETE RESTRICT,
    FOREIGN KEY (match_id, test_case_id)
        REFERENCES matches(id, test_case_id) ON DELETE RESTRICT
);

CREATE TABLE rejudge_batches (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    scope_kind text NOT NULL CHECK (scope_kind IN ('submission', 'task', 'entry', 'game_release', 'contest')),
    scope_id uuid NOT NULL,
    reason text NOT NULL,
    state text NOT NULL CHECK (state IN ('preparing', 'prepared', 'reviewed', 'applying', 'applied', 'cancelled', 'failed')),
    requested_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reviewed_by_user_id uuid REFERENCES users(id) ON DELETE RESTRICT,
    idempotency_key text NOT NULL,
    created_at timestamptz NOT NULL,
    reviewed_at timestamptz,
    applied_at timestamptz,
    cancelled_at timestamptz,
    UNIQUE (contest_id, idempotency_key),
    CHECK (state NOT IN ('reviewed', 'applying', 'applied') OR reviewed_at IS NOT NULL),
    CHECK ((state = 'applied') = (applied_at IS NOT NULL)),
    CHECK ((state = 'cancelled') = (cancelled_at IS NOT NULL))
);

CREATE TABLE rejudge_items (
    rejudge_batch_id uuid NOT NULL REFERENCES rejudge_batches(id) ON DELETE RESTRICT,
    submission_id uuid NOT NULL REFERENCES submissions(id) ON DELETE RESTRICT,
    previous_judgement_id uuid NOT NULL,
    candidate_judgement_id uuid,
    state text NOT NULL CHECK (state IN ('pending', 'ready', 'approved', 'rejected', 'applied')),
    scoring_policy_before_id uuid NOT NULL REFERENCES scoring_policy_versions(id) ON DELETE RESTRICT,
    scoring_policy_after_id uuid REFERENCES scoring_policy_versions(id) ON DELETE RESTRICT,
    PRIMARY KEY (rejudge_batch_id, submission_id),
    UNIQUE (candidate_judgement_id),
    FOREIGN KEY (previous_judgement_id, submission_id)
        REFERENCES judgements(id, submission_id) ON DELETE RESTRICT,
    FOREIGN KEY (candidate_judgement_id, submission_id)
        REFERENCES judgements(id, submission_id) ON DELETE RESTRICT,
    CHECK ((state IN ('ready', 'approved', 'rejected', 'applied')) = (candidate_judgement_id IS NOT NULL)),
    CHECK (candidate_judgement_id IS NULL OR candidate_judgement_id <> previous_judgement_id)
);
