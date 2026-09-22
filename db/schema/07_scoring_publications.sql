SET search_path = agentrix, pg_catalog;

CREATE TABLE score_revisions (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    revision_number bigint NOT NULL CHECK (revision_number > 0),
    state text NOT NULL CHECK (state IN ('building', 'complete', 'failed')),
    source_kind text NOT NULL CHECK (source_kind IN ('full', 'incremental')),
    cutoff_submitted_at timestamptz,
    source_event_through bigint NOT NULL CHECK (source_event_through >= 0),
    created_at timestamptz NOT NULL,
    completed_at timestamptz,
    UNIQUE (contest_id, revision_number),
    UNIQUE (contest_id, id),
    CHECK ((state = 'complete') = (completed_at IS NOT NULL))
);
CREATE UNIQUE INDEX score_revisions_one_building
    ON score_revisions(contest_id) WHERE state = 'building';
CREATE INDEX score_revisions_latest_complete
    ON score_revisions(contest_id, revision_number DESC) WHERE state = 'complete';

CREATE TABLE score_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    contest_id uuid NOT NULL REFERENCES contests(id) ON DELETE RESTRICT,
    event_key text NOT NULL,
    event_kind text NOT NULL CHECK (event_kind IN ('judgement_effective', 'judgement_replaced', 'submission_disposition', 'entry_status')),
    contest_entry_id uuid,
    contest_task_id uuid,
    occurred_at timestamptz NOT NULL,
    applied_revision_id uuid,
    UNIQUE (contest_id, event_key),
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, applied_revision_id)
        REFERENCES score_revisions(contest_id, id) ON DELETE RESTRICT
);
CREATE INDEX score_events_unapplied ON score_events(contest_id, id) WHERE applied_revision_id IS NULL;

CREATE TABLE score_cells (
    revision_id uuid NOT NULL,
    contest_id uuid NOT NULL,
    contest_entry_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    selected_submission_id uuid NOT NULL,
    selected_judgement_id uuid NOT NULL,
    task_points nonnegative_score NOT NULL,
    solved boolean NOT NULL,
    penalty_seconds bigint NOT NULL CHECK (penalty_seconds >= 0),
    wins integer NOT NULL CHECK (wins >= 0),
    draws integer NOT NULL CHECK (draws >= 0),
    losses integer NOT NULL CHECK (losses >= 0),
    computed_at timestamptz NOT NULL,
    PRIMARY KEY (revision_id, contest_entry_id, contest_task_id),
    FOREIGN KEY (contest_id, revision_id)
        REFERENCES score_revisions(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_task_id, selected_submission_id)
        REFERENCES submissions(contest_id, contest_task_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (selected_judgement_id, selected_submission_id)
        REFERENCES judgements(id, submission_id) ON DELETE RESTRICT
);
CREATE INDEX score_cells_entry ON score_cells(revision_id, contest_entry_id);

CREATE TABLE score_rows (
    revision_id uuid NOT NULL,
    contest_id uuid NOT NULL,
    division_id uuid NOT NULL,
    contest_entry_id uuid NOT NULL,
    rank integer NOT NULL CHECK (rank > 0),
    total_points nonnegative_score NOT NULL,
    solved_count integer NOT NULL CHECK (solved_count >= 0),
    total_penalty_seconds bigint NOT NULL CHECK (total_penalty_seconds >= 0),
    total_wins integer NOT NULL CHECK (total_wins >= 0),
    tie_break_value numeric(24, 8) NOT NULL,
    computed_at timestamptz NOT NULL,
    PRIMARY KEY (revision_id, contest_entry_id),
    FOREIGN KEY (contest_id, revision_id)
        REFERENCES score_revisions(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_entry_id, division_id)
        REFERENCES contest_entries(contest_id, id, division_id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, division_id)
        REFERENCES contest_divisions(contest_id, id) ON DELETE RESTRICT
);
CREATE INDEX score_rows_ranking
    ON score_rows(revision_id, division_id, rank, contest_entry_id);

CREATE TABLE scoreboard_publications (
    id uuid PRIMARY KEY,
    contest_id uuid NOT NULL,
    source_revision_id uuid NOT NULL,
    publication_number bigint NOT NULL CHECK (publication_number > 0),
    audience text NOT NULL CHECK (audience IN ('public', 'team', 'jury')),
    kind text NOT NULL CHECK (kind IN ('freeze', 'provisional', 'final', 'unfreeze')),
    cutoff_submitted_at timestamptz,
    published_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    published_at timestamptz NOT NULL,
    UNIQUE (contest_id, audience, publication_number),
    UNIQUE (contest_id, id),
    FOREIGN KEY (contest_id, source_revision_id)
        REFERENCES score_revisions(contest_id, id) ON DELETE RESTRICT,
    CHECK (kind <> 'freeze' OR cutoff_submitted_at IS NOT NULL)
);

CREATE TABLE scoreboard_publication_rows (
    publication_id uuid NOT NULL,
    contest_id uuid NOT NULL,
    division_id uuid NOT NULL,
    contest_entry_id uuid NOT NULL,
    rank integer NOT NULL CHECK (rank > 0),
    total_points nonnegative_score NOT NULL,
    solved_count integer NOT NULL CHECK (solved_count >= 0),
    total_penalty_seconds bigint NOT NULL CHECK (total_penalty_seconds >= 0),
    total_wins integer NOT NULL CHECK (total_wins >= 0),
    tie_break_value numeric(24, 8) NOT NULL,
    PRIMARY KEY (publication_id, contest_entry_id),
    FOREIGN KEY (contest_id, publication_id)
        REFERENCES scoreboard_publications(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_entry_id, division_id)
        REFERENCES contest_entries(contest_id, id, division_id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, division_id)
        REFERENCES contest_divisions(contest_id, id) ON DELETE RESTRICT
);
CREATE INDEX scoreboard_publication_rows_order
    ON scoreboard_publication_rows(publication_id, division_id, rank, contest_entry_id);

CREATE TABLE scoreboard_publication_cells (
    publication_id uuid NOT NULL,
    contest_id uuid NOT NULL,
    contest_entry_id uuid NOT NULL,
    contest_task_id uuid NOT NULL,
    task_points nonnegative_score NOT NULL,
    solved boolean NOT NULL,
    penalty_seconds bigint NOT NULL CHECK (penalty_seconds >= 0),
    PRIMARY KEY (publication_id, contest_entry_id, contest_task_id),
    FOREIGN KEY (contest_id, publication_id)
        REFERENCES scoreboard_publications(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_entry_id)
        REFERENCES contest_entries(contest_id, id) ON DELETE RESTRICT,
    FOREIGN KEY (contest_id, contest_task_id)
        REFERENCES contest_tasks(contest_id, id) ON DELETE RESTRICT
);

CREATE TRIGGER score_revisions_immutable_after_complete
    BEFORE UPDATE OR DELETE ON score_revisions
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_when_sealed();
CREATE TRIGGER scoreboard_publications_immutable
    BEFORE UPDATE OR DELETE ON scoreboard_publications
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER scoreboard_publication_rows_immutable
    BEFORE UPDATE OR DELETE ON scoreboard_publication_rows
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER scoreboard_publication_cells_immutable
    BEFORE UPDATE OR DELETE ON scoreboard_publication_cells
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
