SET search_path = agentrix, pg_catalog;

CREATE TABLE users (
    id uuid PRIMARY KEY,
    username text NOT NULL CHECK (username = btrim(username) AND length(username) BETWEEN 3 AND 64),
    username_normalized text GENERATED ALWAYS AS (lower(btrim(username))) STORED,
    email text NOT NULL CHECK (email = btrim(email) AND position('@' IN email) > 1),
    email_normalized text GENERATED ALWAYS AS (lower(btrim(email))) STORED,
    status text NOT NULL CHECK (status IN ('pending', 'active', 'suspended', 'disabled')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    disabled_at timestamptz,
    CHECK (updated_at >= created_at),
    CHECK ((status = 'disabled') = (disabled_at IS NOT NULL)),
    UNIQUE (username_normalized),
    UNIQUE (email_normalized)
);

CREATE TABLE user_credentials (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    password_hash bytea NOT NULL CHECK (octet_length(password_hash) >= 32),
    algorithm text NOT NULL CHECK (algorithm IN ('argon2id', 'scrypt', 'bcrypt', 'external')),
    algorithm_parameters jsonb NOT NULL CHECK (jsonb_typeof(algorithm_parameters) = 'object'),
    created_at timestamptz NOT NULL,
    retired_at timestamptz,
    CHECK (retired_at IS NULL OR retired_at >= created_at)
);
CREATE UNIQUE INDEX user_credentials_one_active
    ON user_credentials(user_id) WHERE retired_at IS NULL;

CREATE TABLE sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    family_id uuid NOT NULL,
    token_digest sha256_digest NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    last_used_at timestamptz,
    revoked_at timestamptz,
    revoke_reason text,
    rotated_from_session_id uuid UNIQUE REFERENCES sessions(id) ON DELETE RESTRICT,
    CHECK (expires_at > created_at),
    CHECK (last_used_at IS NULL OR last_used_at >= created_at),
    CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL)),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX sessions_active_lookup
    ON sessions(user_id, expires_at) WHERE revoked_at IS NULL;

CREATE TABLE roles (
    id uuid PRIMARY KEY,
    key text NOT NULL UNIQUE CHECK (key ~ '^[a-z][a-z0-9_.-]{1,63}$'),
    scope_kind text NOT NULL CHECK (scope_kind IN ('global', 'contest', 'both')),
    description text NOT NULL
);

CREATE TABLE permissions (
    id uuid PRIMARY KEY,
    key text NOT NULL UNIQUE CHECK (key ~ '^[a-z][a-z0-9_.-]{1,95}$'),
    description text NOT NULL
);

CREATE TABLE role_permissions (
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    permission_id uuid NOT NULL REFERENCES permissions(id) ON DELETE RESTRICT,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    role_id uuid NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    granted_by_user_id uuid REFERENCES users(id) ON DELETE RESTRICT,
    granted_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CHECK (revoked_at IS NULL OR revoked_at >= granted_at),
    PRIMARY KEY (user_id, role_id, granted_at)
);
CREATE UNIQUE INDEX user_roles_one_active
    ON user_roles(user_id, role_id) WHERE revoked_at IS NULL;

CREATE TABLE teams (
    id uuid PRIMARY KEY,
    slug text NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}$'),
    display_name text NOT NULL CHECK (length(btrim(display_name)) BETWEEN 1 AND 128),
    status text NOT NULL CHECK (status IN ('active', 'suspended', 'archived')),
    created_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL
);

CREATE TABLE team_memberships (
    id uuid PRIMARY KEY,
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE RESTRICT,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    member_role text NOT NULL CHECK (member_role IN ('owner', 'administrator', 'member')),
    joined_at timestamptz NOT NULL,
    left_at timestamptz,
    granted_by_user_id uuid REFERENCES users(id) ON DELETE RESTRICT,
    CHECK (left_at IS NULL OR left_at > joined_at),
    UNIQUE (team_id, user_id, joined_at)
);
CREATE UNIQUE INDEX team_memberships_one_active
    ON team_memberships(team_id, user_id) WHERE left_at IS NULL;
CREATE UNIQUE INDEX teams_one_active_owner
    ON team_memberships(team_id) WHERE left_at IS NULL AND member_role = 'owner';
CREATE INDEX team_memberships_authorization
    ON team_memberships(team_id, user_id, joined_at, left_at);

CREATE TABLE audit_events (
    id uuid PRIMARY KEY,
    occurred_at timestamptz NOT NULL,
    actor_user_id uuid REFERENCES users(id) ON DELETE RESTRICT,
    actor_service text,
    action text NOT NULL CHECK (action ~ '^[a-z][a-z0-9_.-]{2,127}$'),
    entity_type text NOT NULL,
    entity_id uuid,
    reason text,
    request_id text,
    summary jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(summary) = 'object'),
    CHECK ((actor_user_id IS NOT NULL)::integer + (actor_service IS NOT NULL)::integer = 1)
);
CREATE INDEX audit_events_entity ON audit_events(entity_type, entity_id, occurred_at);
