SET search_path = agentrix, pg_catalog;

CREATE FUNCTION enforce_artifact_transition()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' OR OLD.publication_state <> 'pending' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'artifact terminal state is immutable';
    END IF;
    IF NEW.publication_state NOT IN ('ready', 'failed') THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'artifact may transition only from pending to ready or failed';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER artifacts_transition
    BEFORE UPDATE OR DELETE ON artifacts
    FOR EACH ROW EXECUTE FUNCTION enforce_artifact_transition();

CREATE FUNCTION validate_game_release_artifacts()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM artifacts WHERE id=NEW.manifest_artifact_id
        AND kind='manifest' AND publication_state='ready' AND sha256=NEW.manifest_digest)
       OR NOT EXISTS (SELECT 1 FROM artifacts WHERE id=NEW.engine_artifact_id
        AND kind='engine' AND publication_state='ready' AND sha256=NEW.engine_digest)
       OR NOT EXISTS (SELECT 1 FROM artifacts WHERE id=NEW.rules_artifact_id
        AND kind='rules' AND publication_state='ready' AND sha256=NEW.rules_digest) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'game release artifacts are not ready or digest-matched';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER game_releases_validate_artifacts
    BEFORE INSERT ON game_releases
    FOR EACH ROW EXECUTE FUNCTION validate_game_release_artifacts();

CREATE FUNCTION validate_baseline_artifact()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM artifacts WHERE id=NEW.executable_artifact_id
        AND kind='baseline' AND publication_state='ready' AND sha256=NEW.executable_digest) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'baseline artifact is not ready or digest-matched';
    END IF;
    IF NEW.protocol_version <> (SELECT protocol_version FROM game_releases WHERE id=NEW.game_release_id) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'baseline protocol differs from game release';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER baseline_programs_validate_artifact
    BEFORE INSERT ON baseline_programs
    FOR EACH ROW EXECUTE FUNCTION validate_baseline_artifact();

CREATE FUNCTION validate_role_assignment_scope()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_scope text;
BEGIN
    SELECT scope_kind INTO v_scope FROM roles WHERE id = NEW.role_id;
    IF TG_TABLE_NAME = 'user_roles' AND v_scope NOT IN ('global', 'both') THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'contest-only role cannot be assigned globally';
    ELSIF TG_TABLE_NAME = 'contest_user_roles' AND v_scope NOT IN ('contest', 'both') THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'global-only role cannot be assigned to a contest';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER user_roles_validate_scope
    BEFORE INSERT OR UPDATE ON user_roles
    FOR EACH ROW EXECUTE FUNCTION validate_role_assignment_scope();
CREATE TRIGGER contest_user_roles_validate_scope
    BEFORE INSERT OR UPDATE ON contest_user_roles
    FOR EACH ROW EXECUTE FUNCTION validate_role_assignment_scope();

CREATE FUNCTION prevent_membership_overlap()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM 1 FROM teams WHERE id = NEW.team_id FOR UPDATE;
    IF EXISTS (
        SELECT 1 FROM team_memberships tm
         WHERE tm.team_id = NEW.team_id AND tm.user_id = NEW.user_id
           AND tm.id <> NEW.id
           AND NEW.joined_at < COALESCE(tm.left_at, 'infinity'::timestamptz)
           AND tm.joined_at < COALESCE(NEW.left_at, 'infinity'::timestamptz)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23P01', MESSAGE = 'team membership periods overlap';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER team_memberships_no_overlap
    BEFORE INSERT OR UPDATE ON team_memberships
    FOR EACH ROW EXECUTE FUNCTION prevent_membership_overlap();

CREATE FUNCTION protect_batch_children()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_batch_id uuid; v_state text;
BEGIN
    v_batch_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.batch_id ELSE NEW.batch_id END;
    SELECT state INTO v_state FROM evaluation_batches WHERE id = v_batch_id FOR SHARE;
    IF v_state <> 'draft' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'sealed batch roster is immutable';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER batch_roster_only_draft
    BEFORE INSERT OR UPDATE OR DELETE ON batch_roster
    FOR EACH ROW EXECUTE FUNCTION protect_batch_children();

CREATE FUNCTION protect_published_task()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.published_at IS NOT NULL AND (TG_OP = 'DELETE' OR NEW IS DISTINCT FROM OLD) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'published contest task is immutable';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER contest_tasks_published_immutable
    BEFORE UPDATE OR DELETE ON contest_tasks
    FOR EACH ROW EXECUTE FUNCTION protect_published_task();

CREATE FUNCTION protect_batch_definition()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.state <> 'draft' AND ROW(
        NEW.contest_id, NEW.contest_task_id, NEW.game_release_id,
        NEW.evaluation_policy_version_id, NEW.scoring_policy_version_id,
        NEW.purpose, NEW.reference_kind, NEW.roster_digest, NEW.created_at
    ) IS DISTINCT FROM ROW(
        OLD.contest_id, OLD.contest_task_id, OLD.game_release_id,
        OLD.evaluation_policy_version_id, OLD.scoring_policy_version_id,
        OLD.purpose, OLD.reference_kind, OLD.roster_digest, OLD.created_at
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'sealed batch definition is immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER evaluation_batches_definition_immutable
    BEFORE UPDATE ON evaluation_batches
    FOR EACH ROW EXECUTE FUNCTION protect_batch_definition();

CREATE FUNCTION protect_match_seats()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_match_id uuid; v_state text;
BEGIN
    v_match_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.match_id ELSE NEW.match_id END;
    SELECT state INTO v_state FROM matches WHERE id = v_match_id FOR SHARE;
    IF v_state <> 'planned' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'sealed match seats are immutable';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER match_seats_only_planned
    BEFORE INSERT OR UPDATE OR DELETE ON match_seats
    FOR EACH ROW EXECUTE FUNCTION protect_match_seats();

CREATE FUNCTION protect_match_definition()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.state <> 'planned' AND ROW(
        NEW.batch_id, NEW.contest_id, NEW.contest_task_id, NEW.game_release_id,
        NEW.test_case_id, NEW.purpose, NEW.seed, NEW.scenario_artifact_id,
        NEW.schedule_key, NEW.specification_digest, NEW.allow_duplicate_programs,
        NEW.created_at, NEW.sealed_at
    ) IS DISTINCT FROM ROW(
        OLD.batch_id, OLD.contest_id, OLD.contest_task_id, OLD.game_release_id,
        OLD.test_case_id, OLD.purpose, OLD.seed, OLD.scenario_artifact_id,
        OLD.schedule_key, OLD.specification_digest, OLD.allow_duplicate_programs,
        OLD.created_at, OLD.sealed_at
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'sealed match definition is immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER matches_definition_immutable
    BEFORE UPDATE ON matches
    FOR EACH ROW EXECUTE FUNCTION protect_match_definition();

CREATE FUNCTION protect_terminal_attempt()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.state <> 'running' AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'terminal match attempt is immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER match_attempts_terminal_immutable
    BEFORE UPDATE ON match_attempts
    FOR EACH ROW EXECUTE FUNCTION protect_terminal_attempt();

CREATE FUNCTION enforce_replay_transition()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' OR OLD.state IN ('published', 'rejected')
       OR (OLD.state = 'verified' AND NEW.state <> 'published')
       OR (OLD.state = 'pending' AND NEW.state NOT IN ('verified', 'rejected')) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'invalid or terminal replay transition';
    END IF;
    IF ROW(NEW.match_id, NEW.attempt_id, NEW.artifact_id, NEW.format,
           NEW.format_version, NEW.duration_ticks)
       IS DISTINCT FROM ROW(OLD.match_id, OLD.attempt_id, OLD.artifact_id,
           OLD.format, OLD.format_version, OLD.duration_ticks) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'replay identity and content are immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER match_replays_transition
    BEFORE UPDATE OR DELETE ON match_replays
    FOR EACH ROW EXECUTE FUNCTION enforce_replay_transition();

CREATE FUNCTION protect_judgement_evidence()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF ROW(NEW.contest_id, NEW.contest_task_id, NEW.submission_id, NEW.batch_id,
           NEW.build_attempt_id, NEW.evaluation_policy_version_id,
           NEW.scoring_policy_version_id, NEW.score_scope, NEW.verdict,
           NEW.engine_score, NEW.task_points, NEW.penalty_seconds,
           NEW.wins, NEW.draws, NEW.losses, NEW.judged_at)
       IS DISTINCT FROM ROW(OLD.contest_id, OLD.contest_task_id, OLD.submission_id,
           OLD.batch_id, OLD.build_attempt_id, OLD.evaluation_policy_version_id,
           OLD.scoring_policy_version_id, OLD.score_scope, OLD.verdict,
           OLD.engine_score, OLD.task_points, OLD.penalty_seconds,
           OLD.wins, OLD.draws, OLD.losses, OLD.judged_at) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'judgement evidence is immutable';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER judgements_evidence_immutable
    BEFORE UPDATE ON judgements
    FOR EACH ROW EXECUTE FUNCTION protect_judgement_evidence();

CREATE TRIGGER audit_events_immutable
    BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER submission_disposition_events_immutable
    BEFORE UPDATE OR DELETE ON submission_disposition_events
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER submissions_no_delete
    BEFORE DELETE ON submissions
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER build_attempts_no_delete
    BEFORE DELETE ON build_attempts
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER match_attempts_no_delete
    BEFORE DELETE ON match_attempts
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER match_seat_results_immutable
    BEFORE UPDATE OR DELETE ON match_seat_results
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER judgement_cases_immutable
    BEFORE UPDATE OR DELETE ON judgement_cases
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER judgements_no_delete
    BEFORE DELETE ON judgements
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER judgement_matches_immutable
    BEFORE UPDATE OR DELETE ON judgement_matches
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();
CREATE TRIGGER contest_events_immutable
    BEFORE UPDATE OR DELETE ON contest_events
    FOR EACH ROW EXECUTE FUNCTION reject_mutation_after_insert();

CREATE FUNCTION validate_judgement_match()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_source_submission uuid; v_accepted_attempt uuid;
BEGIN
    SELECT br.submission_id, m.accepted_attempt_id
      INTO v_source_submission, v_accepted_attempt
      FROM match_seats ms
      JOIN batch_roster br ON br.batch_id = ms.batch_id AND br.id = ms.roster_item_id
      JOIN matches m ON m.id = ms.match_id
     WHERE ms.match_id = NEW.match_id AND ms.id = NEW.seat_id;
    IF v_source_submission IS DISTINCT FROM NEW.submission_id THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'judgement seat does not contain judged submission';
    END IF;
    IF v_accepted_attempt IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'judgement match has no accepted attempt';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER judgement_matches_validate_source
    BEFORE INSERT OR UPDATE ON judgement_matches
    FOR EACH ROW EXECUTE FUNCTION validate_judgement_match();

CREATE FUNCTION protect_score_projection()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE v_revision_id uuid; v_state text;
BEGIN
    v_revision_id := CASE WHEN TG_OP = 'DELETE' THEN OLD.revision_id ELSE NEW.revision_id END;
    SELECT state INTO v_state FROM score_revisions WHERE id = v_revision_id FOR SHARE;
    IF v_state <> 'building' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'completed score revision is immutable';
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER score_cells_only_building
    BEFORE INSERT OR UPDATE OR DELETE ON score_cells
    FOR EACH ROW EXECUTE FUNCTION protect_score_projection();
CREATE TRIGGER score_rows_only_building
    BEFORE INSERT OR UPDATE OR DELETE ON score_rows
    FOR EACH ROW EXECUTE FUNCTION protect_score_projection();
