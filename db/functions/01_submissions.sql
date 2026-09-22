SET search_path = agentrix, pg_catalog;

CREATE FUNCTION register_team_for_contest(
    p_entry_id uuid, p_contest_id uuid, p_team_id uuid, p_division_id uuid,
    p_actor_user_id uuid, p_late_approved_by_user_id uuid DEFAULT NULL
) RETURNS contest_entries
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_now timestamptz := clock_timestamp(); v_contest contests%ROWTYPE;
        v_entry contest_entries%ROWTYPE;
BEGIN
    SELECT * INTO v_entry FROM contest_entries
     WHERE contest_id = p_contest_id AND team_id = p_team_id;
    IF FOUND THEN RETURN v_entry; END IF;
    SELECT * INTO v_contest FROM contests WHERE id = p_contest_id FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contest not found'; END IF;
    IF v_now < v_contest.registration_opens_at THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'registration is not open';
    END IF;
    IF v_now > v_contest.registration_closes_at THEN
        IF v_contest.late_registration_policy = 'reject' THEN
            RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'late registration is rejected';
        END IF;
        IF p_late_approved_by_user_id IS NULL OR NOT EXISTS (
            SELECT 1 FROM contest_user_roles cur JOIN roles r ON r.id = cur.role_id
             WHERE cur.contest_id = p_contest_id AND cur.user_id = p_late_approved_by_user_id
               AND cur.revoked_at IS NULL AND r.key = 'organizer'
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'late registration requires an active contest organizer';
        END IF;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM team_memberships tm JOIN users u ON u.id = tm.user_id
         WHERE tm.team_id = p_team_id AND tm.user_id = p_actor_user_id AND u.status = 'active'
           AND tm.joined_at <= v_now AND (tm.left_at IS NULL OR tm.left_at > v_now)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'actor is not an active team member';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM contest_divisions WHERE id = p_division_id AND contest_id = p_contest_id) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'division belongs to another contest';
    END IF;
    INSERT INTO contest_entries(id, contest_id, team_id, division_id, status,
                                eligibility, registered_at)
    VALUES (p_entry_id, p_contest_id, p_team_id, p_division_id, 'pending',
            'pending_review', v_now)
    RETURNING * INTO v_entry;
    RETURN v_entry;
END;
$$;

CREATE FUNCTION transition_contest_state(
    p_contest_id uuid, p_new_state text, p_actor_user_id uuid,
    p_audit_event_id uuid, p_reason text
) RETURNS contests
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_contest contests%ROWTYPE; v_now timestamptz := clock_timestamp(); v_allowed boolean;
BEGIN
    SELECT * INTO v_contest FROM contests WHERE id = p_contest_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contest not found'; END IF;
    IF v_contest.state = p_new_state THEN RETURN v_contest; END IF;
    v_allowed := CASE v_contest.state
      WHEN 'draft' THEN p_new_state IN ('registration', 'cancelled')
      WHEN 'registration' THEN p_new_state IN ('submission', 'cancelled')
      WHEN 'submission' THEN p_new_state IN ('running', 'cancelled')
      WHEN 'running' THEN p_new_state IN ('frozen', 'closed', 'cancelled')
      WHEN 'frozen' THEN p_new_state IN ('running', 'closed', 'cancelled')
      WHEN 'closed' THEN p_new_state = 'published'
      ELSE false END;
    IF NOT v_allowed THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'invalid contest state transition';
    END IF;
    IF p_new_state = 'registration' AND v_now < v_contest.registration_opens_at
       OR p_new_state = 'submission' AND v_now < v_contest.submission_opens_at
       OR p_new_state = 'running' AND v_now < v_contest.starts_at
       OR p_new_state = 'frozen' AND (v_contest.freeze_at IS NULL OR v_now < v_contest.freeze_at)
       OR p_new_state = 'closed' AND v_now < v_contest.ends_at
       OR p_new_state = 'published' AND (v_contest.unfreeze_at IS NOT NULL AND v_now < v_contest.unfreeze_at) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'contest transition is earlier than its configured time';
    END IF;
    UPDATE contests SET state = p_new_state,
           published_at = CASE WHEN p_new_state = 'published' THEN v_now ELSE published_at END
     WHERE id = p_contest_id RETURNING * INTO v_contest;
    INSERT INTO audit_events(id, occurred_at, actor_user_id, action, entity_type,
                             entity_id, reason, summary)
    VALUES (p_audit_event_id, v_now, p_actor_user_id, 'contest.state_changed',
            'contest', p_contest_id, p_reason,
            jsonb_build_object('new_state', p_new_state));
    RETURN v_contest;
END;
$$;

CREATE FUNCTION submit_program(
    p_submission_id uuid,
    p_contest_entry_id uuid,
    p_contest_task_id uuid,
    p_actor_user_id uuid,
    p_source_artifact_id uuid,
    p_toolchain_id uuid,
    p_idempotency_key text
) RETURNS submissions
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE
    v_now timestamptz := clock_timestamp();
    v_contest contests%ROWTYPE;
    v_entry contest_entries%ROWTYPE;
    v_task contest_tasks%ROWTYPE;
    v_existing submissions%ROWTYPE;
    v_limit integer;
    v_source_size bigint;
    v_source_state text;
    v_language_limit bigint;
BEGIN
    SELECT s.* INTO v_existing
      FROM submissions s
     WHERE s.contest_entry_id = p_contest_entry_id
       AND s.contest_task_id = p_contest_task_id
       AND s.idempotency_key = p_idempotency_key;
    IF FOUND THEN RETURN v_existing; END IF;

    SELECT * INTO v_entry FROM contest_entries WHERE id = p_contest_entry_id FOR SHARE;
    IF NOT FOUND OR v_entry.status <> 'accepted' THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'entry is not accepted';
    END IF;
    SELECT * INTO v_task FROM contest_tasks WHERE id = p_contest_task_id FOR SHARE;
    IF NOT FOUND OR v_task.contest_id <> v_entry.contest_id THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'entry and task belong to different contests';
    END IF;
    SELECT * INTO v_contest FROM contests WHERE id = v_entry.contest_id FOR SHARE;
    IF NOT v_task.allow_submit OR v_now < v_contest.submission_opens_at OR v_now > v_contest.submission_closes_at THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'submission window is closed';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM users u WHERE u.id = p_actor_user_id AND u.status = 'active')
       OR NOT EXISTS (
        SELECT 1 FROM team_memberships tm
         WHERE tm.team_id = v_entry.team_id AND tm.user_id = p_actor_user_id
           AND tm.joined_at <= v_now AND (tm.left_at IS NULL OR tm.left_at > v_now)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'actor is not an active member of the entry team';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM task_toolchains tt
         WHERE tt.task_id = p_contest_task_id AND tt.toolchain_id = p_toolchain_id AND tt.enabled
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'toolchain is not enabled for task';
    END IF;
    SELECT a.size_bytes, a.publication_state, l.source_size_limit_bytes
      INTO v_source_size, v_source_state, v_language_limit
      FROM artifacts a
      JOIN toolchains tc ON tc.id = p_toolchain_id
      JOIN languages l ON l.id = tc.language_id
     WHERE a.id = p_source_artifact_id AND a.kind = 'source';
    IF NOT FOUND OR v_source_state <> 'ready' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'source artifact is not ready';
    END IF;
    IF v_source_size > v_language_limit THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'source artifact exceeds language size limit';
    END IF;
    v_limit := COALESCE(v_task.submission_limit, v_contest.default_submission_limit_per_task);
    IF v_limit IS NOT NULL AND (
        SELECT count(*) FROM submissions
         WHERE contest_entry_id = p_contest_entry_id AND contest_task_id = p_contest_task_id
    ) >= v_limit THEN
        RAISE EXCEPTION USING ERRCODE = '54000', MESSAGE = 'submission limit reached';
    END IF;

    INSERT INTO submissions(
        id, contest_id, contest_entry_id, contest_task_id, submitted_by_user_id,
        source_artifact_id, toolchain_id, submitted_at, received_at,
        idempotency_key, disposition
    ) VALUES (
        p_submission_id, v_entry.contest_id, p_contest_entry_id, p_contest_task_id,
        p_actor_user_id, p_source_artifact_id, p_toolchain_id, v_now, v_now,
        p_idempotency_key, 'eligible'
    ) RETURNING * INTO v_existing;
    RETURN v_existing;
END;
$$;

CREATE FUNCTION change_submission_disposition(
    p_event_id uuid, p_submission_id uuid, p_new_disposition text,
    p_reason text, p_actor_user_id uuid
) RETURNS submissions
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_submission submissions%ROWTYPE; v_old text; v_now timestamptz := clock_timestamp();
BEGIN
    IF p_new_disposition NOT IN ('eligible', 'ignored', 'withdrawn', 'disqualified') OR btrim(p_reason) = '' THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'invalid disposition or empty reason';
    END IF;
    SELECT * INTO v_submission FROM submissions WHERE id = p_submission_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'submission not found'; END IF;
    v_old := v_submission.disposition;
    IF v_old = p_new_disposition THEN RETURN v_submission; END IF;
    UPDATE submissions SET disposition = p_new_disposition, disposition_reason =
        CASE WHEN p_new_disposition = 'eligible' THEN NULL ELSE p_reason END
     WHERE id = p_submission_id RETURNING * INTO v_submission;
    INSERT INTO submission_disposition_events
        (id, submission_id, previous_disposition, new_disposition, reason, changed_by_user_id, changed_at)
    VALUES (p_event_id, p_submission_id, v_old, p_new_disposition, p_reason, p_actor_user_id, v_now);
    INSERT INTO score_events(contest_id, event_key, event_kind, contest_entry_id, contest_task_id, occurred_at)
    VALUES (v_submission.contest_id, 'disposition:' || p_event_id, 'submission_disposition',
            v_submission.contest_entry_id, v_submission.contest_task_id, v_now)
    ON CONFLICT (contest_id, event_key) DO NOTHING;
    RETURN v_submission;
END;
$$;
