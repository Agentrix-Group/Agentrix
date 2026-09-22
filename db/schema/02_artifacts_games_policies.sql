SET search_path = agentrix, pg_catalog;

CREATE TABLE artifacts (
    id uuid PRIMARY KEY,
    kind text NOT NULL CHECK (kind IN (
        'source', 'executable', 'manifest', 'engine', 'rules', 'scenario',
        'baseline', 'build_log', 'match_log', 'metrics', 'feedback', 'replay'
    )),
    publication_state text NOT NULL CHECK (publication_state IN ('pending', 'ready', 'failed')),
    sha256 sha256_digest,
    size_bytes bigint CHECK (size_bytes IS NULL OR size_bytes BETWEEN 0 AND 1099511627776),
    media_type text,
    storage_key text,
    created_at timestamptz NOT NULL,
    verified_at timestamptz,
    failure_reason text,
    CHECK (
        (publication_state = 'pending' AND sha256 IS NULL AND verified_at IS NULL AND failure_reason IS NULL)
        OR (publication_state = 'ready' AND sha256 IS NOT NULL AND size_bytes IS NOT NULL
            AND storage_key IS NOT NULL AND verified_at IS NOT NULL AND failure_reason IS NULL)
        OR (publication_state = 'failed' AND verified_at IS NULL AND failure_reason IS NOT NULL)
    ),
    CHECK (verified_at IS NULL OR verified_at >= created_at),
    CHECK (publication_state <> 'ready' OR storage_key <> '')
);
CREATE UNIQUE INDEX artifacts_ready_content
    ON artifacts(sha256, size_bytes) WHERE publication_state = 'ready';
CREATE INDEX artifacts_pending_reconciliation
    ON artifacts(created_at) WHERE publication_state = 'pending';

CREATE TABLE artifact_outbox (
    id uuid PRIMARY KEY,
    artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    operation text NOT NULL CHECK (operation IN ('verify', 'publish', 'delete_orphan')),
    idempotency_key text NOT NULL UNIQUE,
    available_at timestamptz NOT NULL,
    processed_at timestamptz,
    attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count BETWEEN 0 AND 100),
    last_error text
);
CREATE INDEX artifact_outbox_ready
    ON artifact_outbox(available_at, id) WHERE processed_at IS NULL;

CREATE TABLE games (
    id uuid PRIMARY KEY,
    slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
    display_name text NOT NULL,
    description text NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'retired')),
    created_at timestamptz NOT NULL
);

CREATE TABLE game_releases (
    id uuid PRIMARY KEY,
    game_id uuid NOT NULL REFERENCES games(id) ON DELETE RESTRICT,
    version text NOT NULL CHECK (length(btrim(version)) BETWEEN 1 AND 64),
    protocol_version text NOT NULL,
    min_players smallint NOT NULL CHECK (min_players >= 1),
    max_players smallint NOT NULL CHECK (max_players >= min_players AND max_players <= 256),
    manifest_artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    engine_artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    rules_artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    manifest_digest sha256_digest NOT NULL,
    engine_digest sha256_digest NOT NULL,
    rules_digest sha256_digest NOT NULL,
    published_at timestamptz NOT NULL,
    UNIQUE (game_id, version),
    UNIQUE (id, game_id)
);
CREATE TRIGGER game_releases_immutable
    BEFORE UPDATE OR DELETE ON game_releases
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();

CREATE TABLE evaluation_policy_versions (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('fixed_cases', 'reference_opponents', 'round_robin', 'league_schedule')),
    algorithm_version integer NOT NULL CHECK (algorithm_version > 0),
    parameters jsonb NOT NULL CHECK (jsonb_typeof(parameters) = 'object'),
    parameters_digest sha256_digest NOT NULL,
    cpu_limit_ms integer NOT NULL CHECK (cpu_limit_ms BETWEEN 1 AND 86400000),
    memory_limit_bytes bigint NOT NULL CHECK (memory_limit_bytes BETWEEN 1048576 AND 1099511627776),
    tick_limit bigint NOT NULL CHECK (tick_limit > 0),
    max_match_attempts smallint NOT NULL CHECK (max_match_attempts BETWEEN 1 AND 20),
    lease_duration_seconds integer NOT NULL CHECK (lease_duration_seconds BETWEEN 1 AND 86400),
    created_at timestamptz NOT NULL,
    CHECK (valid_evaluation_policy(kind, parameters)),
    UNIQUE (kind, algorithm_version, parameters_digest)
);
CREATE TRIGGER evaluation_policy_versions_immutable
    BEFORE UPDATE OR DELETE ON evaluation_policy_versions
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();

CREATE TABLE scoring_policy_versions (
    id uuid PRIMARY KEY,
    name text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('icpc_pass_fail', 'best_score', 'aggregate_points', 'league_points')),
    algorithm_version integer NOT NULL CHECK (algorithm_version > 0),
    parameters jsonb NOT NULL CHECK (jsonb_typeof(parameters) = 'object'),
    parameters_digest sha256_digest NOT NULL,
    score_unit text NOT NULL CHECK (score_unit IN ('task_points', 'wins', 'league_points')),
    disqualification_task_points nonnegative_score NOT NULL,
    tie_method text NOT NULL CHECK (tie_method IN ('rank', 'dense_rank')),
    created_at timestamptz NOT NULL,
    CHECK (valid_scoring_policy(kind, parameters)),
    CHECK ((kind = 'league_points' AND score_unit = 'league_points')
        OR (kind <> 'league_points' AND score_unit = 'task_points')),
    UNIQUE (kind, algorithm_version, parameters_digest)
);
CREATE TRIGGER scoring_policy_versions_immutable
    BEFORE UPDATE OR DELETE ON scoring_policy_versions
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();

CREATE TABLE languages (
    id uuid PRIMARY KEY,
    key text NOT NULL UNIQUE CHECK (key ~ '^[a-z0-9][a-z0-9_+.-]{0,31}$'),
    display_name text NOT NULL,
    source_size_limit_bytes bigint NOT NULL CHECK (source_size_limit_bytes BETWEEN 1 AND 1073741824),
    active boolean NOT NULL
);

CREATE TABLE toolchains (
    id uuid PRIMARY KEY,
    language_id uuid NOT NULL REFERENCES languages(id) ON DELETE RESTRICT,
    version text NOT NULL,
    image_digest sha256_digest NOT NULL,
    command_contract_digest sha256_digest NOT NULL,
    active boolean NOT NULL,
    UNIQUE (language_id, version)
);

CREATE TABLE baseline_programs (
    id uuid PRIMARY KEY,
    game_release_id uuid NOT NULL REFERENCES game_releases(id) ON DELETE RESTRICT,
    name text NOT NULL,
    version text NOT NULL,
    executable_artifact_id uuid NOT NULL REFERENCES artifacts(id) ON DELETE RESTRICT,
    executable_digest sha256_digest NOT NULL,
    protocol_version text NOT NULL,
    published_at timestamptz NOT NULL,
    UNIQUE (game_release_id, name, version),
    UNIQUE (id, game_release_id)
);
CREATE TRIGGER baseline_programs_immutable
    BEFORE UPDATE OR DELETE ON baseline_programs
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
