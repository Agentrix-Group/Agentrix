SET search_path = agentrix, pg_catalog;

CREATE FUNCTION verify_judgement(
    p_judgement_id uuid, p_juror_user_id uuid, p_verified_at timestamptz
) RETURNS judgements
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_judgement judgements%ROWTYPE;
BEGIN
    SELECT * INTO v_judgement FROM judgements WHERE id = p_judgement_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'judgement not found'; END IF;
    IF v_judgement.state = 'verified' THEN RETURN v_judgement; END IF;
    IF v_judgement.state <> 'awaiting_verification' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'judgement is not awaiting jury verification';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM contest_user_roles cur JOIN roles r ON r.id = cur.role_id
         WHERE cur.contest_id = v_judgement.contest_id AND cur.user_id = p_juror_user_id
           AND cur.revoked_at IS NULL AND r.key IN ('jury', 'organizer')
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501', MESSAGE = 'actor is not an active contest juror';
    END IF;
    UPDATE judgements SET state = 'verified', verified_by_user_id = p_juror_user_id,
           verified_at = p_verified_at WHERE id = p_judgement_id RETURNING * INTO v_judgement;
    RETURN v_judgement;
END;
$$;

CREATE FUNCTION make_judgement_effective(
    p_judgement_id uuid, p_effective_at timestamptz, p_event_key text
) RETURNS judgements
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_judgement judgements%ROWTYPE; v_submission submissions%ROWTYPE;
        v_batch evaluation_batches%ROWTYPE; v_build_state text; v_links integer;
BEGIN
    SELECT * INTO v_judgement FROM judgements WHERE id = p_judgement_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'judgement not found'; END IF;
    IF v_judgement.effective THEN RETURN v_judgement; END IF;
    IF v_judgement.state NOT IN ('complete', 'verified') THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'judgement is not complete or jury verified';
    END IF;
    SELECT * INTO v_submission FROM submissions WHERE id = v_judgement.submission_id FOR SHARE;
    IF v_submission.disposition <> 'eligible' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'submission is not eligible';
    END IF;
    SELECT * INTO v_batch FROM evaluation_batches WHERE id = v_judgement.batch_id FOR SHARE;
    IF v_judgement.evaluation_policy_version_id <> v_batch.evaluation_policy_version_id THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'judgement evaluation policy differs from its batch';
    END IF;
    IF v_judgement.score_scope = 'official' AND v_batch.purpose NOT IN ('official', 'rejudge') THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'official judgement requires an official or rejudge batch';
    END IF;
    SELECT count(*) INTO v_links FROM judgement_matches WHERE judgement_id = p_judgement_id;
    IF v_judgement.verdict = 'compile_error' THEN
        SELECT state INTO v_build_state FROM build_attempts WHERE id = v_judgement.build_attempt_id;
        IF v_build_state IS DISTINCT FROM 'failed' OR v_links <> 0 THEN
            RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'compile-error judgement requires a failed build and no matches';
        END IF;
    ELSIF v_judgement.verdict <> 'infrastructure_error' AND v_links = 0 THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'competitive judgement requires evidence from at least one match';
    END IF;
    IF v_judgement.task_points > (SELECT max_task_points FROM contest_tasks WHERE id = v_judgement.contest_task_id) THEN
        RAISE EXCEPTION USING ERRCODE = '22003', MESSAGE = 'judgement task points exceed task maximum';
    END IF;
    IF EXISTS (
        SELECT 1 FROM judgements j
         WHERE j.submission_id = v_judgement.submission_id
           AND j.contest_task_id = v_judgement.contest_task_id
           AND j.score_scope = 'official' AND j.effective
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'submission already has an effective official judgement';
    END IF;
    UPDATE judgements SET effective = true, effective_at = p_effective_at, superseded_at = NULL
     WHERE id = p_judgement_id RETURNING * INTO v_judgement;
    INSERT INTO score_events(contest_id, event_key, event_kind, contest_entry_id, contest_task_id, occurred_at)
    VALUES (v_judgement.contest_id, p_event_key, 'judgement_effective',
            v_submission.contest_entry_id, v_judgement.contest_task_id, p_effective_at)
    ON CONFLICT (contest_id, event_key) DO NOTHING;
    RETURN v_judgement;
END;
$$;

CREATE FUNCTION apply_rejudge_batch(
    p_rejudge_batch_id uuid, p_applied_at timestamptz
) RETURNS rejudge_batches
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_batch rejudge_batches%ROWTYPE; v_item rejudge_items%ROWTYPE;
        v_previous judgements%ROWTYPE; v_candidate judgements%ROWTYPE;
        v_submission submissions%ROWTYPE;
BEGIN
    SELECT * INTO v_batch FROM rejudge_batches WHERE id = p_rejudge_batch_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'rejudge batch not found'; END IF;
    IF v_batch.state = 'applied' THEN RETURN v_batch; END IF;
    IF v_batch.state <> 'reviewed' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'rejudge batch must be reviewed before applying';
    END IF;
    IF EXISTS (SELECT 1 FROM rejudge_items WHERE rejudge_batch_id = p_rejudge_batch_id AND state <> 'approved')
       OR NOT EXISTS (SELECT 1 FROM rejudge_items WHERE rejudge_batch_id = p_rejudge_batch_id) THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'all rejudge items must be approved';
    END IF;
    PERFORM s.id
      FROM submissions s JOIN rejudge_items ri ON ri.submission_id = s.id
     WHERE ri.rejudge_batch_id = p_rejudge_batch_id
     ORDER BY s.id FOR UPDATE OF s;
    UPDATE rejudge_batches SET state = 'applying' WHERE id = p_rejudge_batch_id;

    FOR v_item IN
        SELECT * FROM rejudge_items WHERE rejudge_batch_id = p_rejudge_batch_id ORDER BY submission_id
    LOOP
        SELECT * INTO v_previous FROM judgements WHERE id = v_item.previous_judgement_id FOR UPDATE;
        SELECT * INTO v_candidate FROM judgements WHERE id = v_item.candidate_judgement_id FOR UPDATE;
        IF NOT v_previous.effective OR v_previous.submission_id <> v_item.submission_id THEN
            RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'previous judgement is no longer effective';
        END IF;
        IF v_candidate.effective OR v_candidate.submission_id <> v_item.submission_id
           OR v_candidate.state NOT IN ('complete', 'verified') THEN
            RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'invalid rejudge candidate';
        END IF;
        IF v_previous.contest_id <> v_batch.contest_id OR v_candidate.contest_id <> v_batch.contest_id
           OR v_candidate.contest_task_id <> v_previous.contest_task_id
           OR v_previous.scoring_policy_version_id <> v_item.scoring_policy_before_id
           OR v_candidate.scoring_policy_version_id IS DISTINCT FROM v_item.scoring_policy_after_id THEN
            RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'rejudge item policy, contest, or task mismatch';
        END IF;
        IF v_candidate.verdict = 'compile_error' THEN
            IF (SELECT state FROM build_attempts WHERE id = v_candidate.build_attempt_id) IS DISTINCT FROM 'failed'
               OR EXISTS (SELECT 1 FROM judgement_matches WHERE judgement_id = v_candidate.id) THEN
                RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'invalid compile-error rejudge evidence';
            END IF;
        ELSIF v_candidate.verdict <> 'infrastructure_error'
              AND NOT EXISTS (SELECT 1 FROM judgement_matches WHERE judgement_id = v_candidate.id) THEN
            RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'rejudge candidate has no match evidence';
        END IF;
        IF v_candidate.task_points > (SELECT max_task_points FROM contest_tasks WHERE id = v_candidate.contest_task_id) THEN
            RAISE EXCEPTION USING ERRCODE = '22003', MESSAGE = 'rejudge task points exceed task maximum';
        END IF;
        SELECT * INTO v_submission FROM submissions WHERE id = v_item.submission_id;
        UPDATE judgements SET effective = false, superseded_at = p_applied_at
         WHERE id = v_previous.id;
        UPDATE judgements SET effective = true, effective_at = p_applied_at, superseded_at = NULL
         WHERE id = v_candidate.id;
        UPDATE rejudge_items SET state = 'applied' WHERE rejudge_batch_id = p_rejudge_batch_id
             AND submission_id = v_item.submission_id;
        INSERT INTO score_events(contest_id, event_key, event_kind, contest_entry_id, contest_task_id, occurred_at)
        VALUES (v_batch.contest_id,
                'rejudge:' || p_rejudge_batch_id || ':' || v_item.submission_id,
                'judgement_replaced', v_submission.contest_entry_id,
                v_submission.contest_task_id, p_applied_at);
    END LOOP;
    UPDATE rejudge_batches SET state = 'applied', applied_at = p_applied_at
     WHERE id = p_rejudge_batch_id RETURNING * INTO v_batch;
    RETURN v_batch;
END;
$$;

CREATE FUNCTION cancel_rejudge_batch(
    p_rejudge_batch_id uuid, p_cancelled_at timestamptz
) RETURNS rejudge_batches
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_batch rejudge_batches%ROWTYPE;
BEGIN
    SELECT * INTO v_batch FROM rejudge_batches WHERE id = p_rejudge_batch_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'rejudge batch not found'; END IF;
    IF v_batch.state = 'cancelled' THEN RETURN v_batch; END IF;
    IF v_batch.state IN ('applying', 'applied') THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'applied rejudge cannot be cancelled';
    END IF;
    UPDATE rejudge_batches SET state = 'cancelled', cancelled_at = p_cancelled_at
     WHERE id = p_rejudge_batch_id RETURNING * INTO v_batch;
    RETURN v_batch;
END;
$$;
