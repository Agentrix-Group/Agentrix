SET search_path = agentrix, pg_catalog;

CREATE TABLE contests (
    id uuid PRIMARY KEY,
    slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
    title text NOT NULL CHECK (length(btrim(title)) BETWEEN 1 AND 160),
    state text NOT NULL CHECK (state IN ('draft', 'registration', 'submission', 'running', 'frozen', 'closed', 'published', 'cancelled')),
    registration_opens_at timestamptz NOT NULL,
    registration_closes_at timestamptz NOT NULL,
    submission_opens_at timestamptz NOT NULL,
    starts_at timestamptz NOT NULL,
    freeze_at timestamptz,
    submission_closes_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    unfreeze_at timestamptz,
    published_at timestamptz,
    late_registration_policy text NOT NULL CHECK (late_registration_policy IN ('reject', 'permission_required')),
    entry_uniqueness_policy text NOT NULL CHECK (entry_uniqueness_policy = 'one_per_team'),
    default_submission_limit_per_task integer CHECK (default_submission_limit_per_task > 0),
    code_visibility text NOT NULL CHECK (code_visibility IN ('team_only', 'jury_only', 'public_after_close')),
    private_feedback_policy text NOT NULL CHECK (private_feedback_policy IN ('hidden', 'verdict_only', 'configured_metrics')),
    rank_method text NOT NULL CHECK (rank_method IN ('rank', 'dense_rank')),
    created_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    CHECK (registration_opens_at < registration_closes_at),
    CHECK (registration_closes_at <= submission_closes_at),
    CHECK (submission_opens_at <= starts_at),
    CHECK (starts_at < ends_at),
    CHECK (submission_opens_at < submission_closes_at),
    CHECK (submission_closes_at <= ends_at),
    CHECK (freeze_at IS NULL OR freeze_at BETWEEN starts_at AND ends_at),
    CHECK (unfreeze_at IS NULL OR (freeze_at IS NOT NULL AND unfreeze_at >= freeze_at)),
    CHECK (published_at IS NULL OR published_at >= starts_at)
);

CREATE TABLE contest_divisions (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    key text NOT NULL CHECK (key ~ '^[a-z0-9][a-z0-9_-]{0,31}$'),
    title text NOT NULL,
    sort_order integer NOT NULL CHECK (sort_order >= 0),
    award_eligible boolean NOT NULL,
    UNIQUE (contest_id, key),
    UNIQUE (contest_id, id)
);

CREATE TABLE contest_entries (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
    division_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('pending', 'accepted', 'rejected', 'withdrawn', 'disqualified')),
    eligibility text NOT NULL CHECK (eligibility IN ('eligible', 'ineligible', 'pending_review')),
    registered_at timestamptz NOT NULL,
    accepted_at timestamptz,
    disqualification_reason text,
    UNIQUE (contest_id, team_id),
    UNIQUE (contest_id, id),
    UNIQUE (contest_id, id, division_id),
    FOREIGN KEY (contest_id, division_id)
        REFERENCES contest_divisions(contest_id, id) ON DELETE RESTRICT,
    CHECK (status NOT IN ('accepted', 'disqualified') OR accepted_at IS NOT NULL),
    CHECK (status NOT IN ('pending', 'rejected') OR accepted_at IS NULL),
    CHECK ((status = 'disqualified') = (disqualification_reason IS NOT NULL))
);
CREATE INDEX contest_entries_ranking
    ON contest_entries(contest_id, division_id, eligibility, status);

CREATE TABLE contest_user_roles (
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    granted_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    granted_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (revoked_at IS NULL OR revoked_at >= granted_at),
    PRIMARY KEY (contest_id, user_id, role_id, granted_at)
);
CREATE UNIQUE INDEX contest_user_roles_one_active
    ON contest_user_roles(contest_id, user_id, role_id) WHERE revoked_at IS NULL;

CREATE TABLE contest_tasks (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    code text NOT NULL CHECK (code ~ '^[A-Z0-9][A-Z0-9_-]{0,15}$'),
    title text NOT NULL,
    position integer NOT NULL CHECK (position > 0),
    game_release_id uuid NOT NULL REFERENCES game_releases(id) ON DELETE RESTRICT,
    evaluation_policy_version_id uuid NOT NULL REFERENCES evaluation_policy_versions(id) ON DELETE RESTRICT,
    scoring_policy_version_id uuid NOT NULL REFERENCES scoring_policy_versions(id) ON DELETE RESTRICT,
    max_task_points nonnegative_score NOT NULL,
    submission_limit integer CHECK (submission_limit > 0),
    allow_submit boolean NOT NULL,
    allow_judge boolean NOT NULL,
    requires_verified_replay boolean NOT NULL,
    visibility text NOT NULL CHECK (visibility IN ('private', 'registered', 'public')),
    published_at timestamptz,
    UNIQUE (contest_id, code),
    UNIQUE (contest_id, position),
    UNIQUE (contest_id, id),
    UNIQUE (contest_id, id, game_release_id, evaluation_policy_version_id, scoring_policy_version_id)
);
CREATE INDEX contest_tasks_release ON contest_tasks(game_release_id);

CREATE TABLE task_toolchains (
    contest_id uuid NOT NULL,
    task_id uuid NOT NULL,
    toolchain_id uuid NOT NULL REFERENCES toolchains(id) ON DELETE RESTRICT,
    enabled boolean NOT NULL,
    PRIMARY KEY (task_id, toolchain_id),
    FOREIGN KEY (contest_id, task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT
);

CREATE TABLE test_groups (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL,
    task_id uuid NOT NULL,
    key text NOT NULL,
    weight nonnegative_score NOT NULL CHECK (weight > 0),
    aggregation text NOT NULL CHECK (aggregation IN ('minimum', 'maximum', 'sum', 'average', 'all_or_nothing')),
    visibility text NOT NULL CHECK (visibility IN ('public', 'private')),
    UNIQUE (task_id, key),
    UNIQUE (task_id, id),
    FOREIGN KEY (contest_id, task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT
);

CREATE TABLE test_cases (
    id uuid PRIMARY KEY,
    task_id uuid NOT NULL,
    group_id uuid NOT NULL,
    key text NOT NULL,
    seed bigint NOT NULL,
    scenario_artifact_id uuid REFERENCES artifacts(id) ON DELETE RESTRICT,
    weight nonnegative_score NOT NULL CHECK (weight > 0),
    visibility text NOT NULL CHECK (visibility IN ('public', 'private')),
    UNIQUE (task_id, key),
    UNIQUE (task_id, id),
    FOREIGN KEY (task_id, group_id)
        REFERENCES test_groups(task_id, id) ON DELETE RESTRICT
);

CREATE TABLE task_baselines (
    task_id uuid NOT NULL REFERENCES contest_tasks(id) ON DELETE RESTRICT,
    baseline_program_id uuid NOT NULL REFERENCES baseline_programs(id) ON DELETE RESTRICT,
    purpose text NOT NULL CHECK (purpose IN ('reference', 'opponent', 'validation')),
    PRIMARY KEY (task_id, baseline_program_id, purpose)
);
