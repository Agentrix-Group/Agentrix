SET search_path = agentrix, pg_catalog;

CREATE TABLE clarification_threads (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL,
    contest_entry_id uuid NOT NULL,
    contest_task_id uuid,
    subject text NOT NULL,
    state text NOT NULL CHECK (state IN ('open', 'answered', 'closed')),
    created_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL,
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT
);

CREATE TABLE clarification_messages (
    id uuid PRIMARY KEY,
    thread_id uuid NOT NULL REFERENCES clarification_threads(id) ON DELETE RESTRICT,
    author_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    audience text NOT NULL CHECK (audience IN ('requesting_team', 'jury', 'all_entries', 'public')),
    body text NOT NULL CHECK (length(btrim(body)) BETWEEN 1 AND 20000),
    created_at timestamptz NOT NULL,
    published_at timestamptz
);

CREATE TABLE contest_announcements (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    author_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    audience text NOT NULL CHECK (audience IN ('entries', 'jury', 'public')),
    title text NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL,
    published_at timestamptz
);

CREATE TABLE contest_awards (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL,
    contest_entry_id uuid NOT NULL,
    source_publication_id uuid NOT NULL,
    key text NOT NULL,
    title text NOT NULL,
    awarded_at timestamptz NOT NULL,
    UNIQUE (contest_id, key, contest_entry_id),
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, source_publication_id)
        REFERENCES scoreboard_publications(contest_id, id) ON DELETE RESTRICT
);

CREATE TABLE contest_events (
    sequence_number bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    event_key text NOT NULL,
    event_kind text NOT NULL,
    audience text NOT NULL CHECK (audience IN ('team', 'jury', 'public')),
    entity_id uuid,
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload) = 'object'),
    occurred_at timestamptz NOT NULL,
    UNIQUE (contest_id, event_key)
);
CREATE INDEX contest_events_stream
    ON contest_events(contest_id, audience, sequence_number);

