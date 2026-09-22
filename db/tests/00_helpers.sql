CREATE SCHEMA test_support;

CREATE FUNCTION test_support.assert_true(p_condition boolean, p_message text)
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    IF NOT COALESCE(p_condition, false) THEN
        RAISE EXCEPTION 'assertion failed: %', p_message;
    END IF;
END;
$$;

CREATE FUNCTION test_support.expect_error(
    p_statement text, p_expected_state text, p_message_fragment text
) RETURNS void LANGUAGE plpgsql AS $$
DECLARE v_state text; v_message text;
BEGIN
    BEGIN
        EXECUTE p_statement;
    EXCEPTION WHEN OTHERS THEN
        GET STACKED DIAGNOSTICS v_state = RETURNED_SQLSTATE, v_message = MESSAGE_TEXT;
        IF p_expected_state IS NOT NULL AND v_state <> p_expected_state THEN
            RAISE EXCEPTION 'expected SQLSTATE %, got %: %', p_expected_state, v_state, v_message;
        END IF;
        IF p_message_fragment IS NOT NULL AND position(p_message_fragment IN v_message) = 0 THEN
            RAISE EXCEPTION 'expected error containing %, got %', p_message_fragment, v_message;
        END IF;
        RETURN;
    END;
    RAISE EXCEPTION 'expected statement to fail: %', p_statement;
END;
$$;

CREATE FUNCTION test_support.explain_text(p_statement text)
RETURNS text LANGUAGE plpgsql AS $$
DECLARE v_line text; v_result text := '';
BEGIN
    FOR v_line IN EXECUTE 'EXPLAIN ' || p_statement LOOP
        v_result := v_result || E'\n' || v_line;
    END LOOP;
    RETURN v_result;
END;
$$;

CREATE FUNCTION test_support.complete_fixture_match()
RETURNS uuid LANGUAGE plpgsql AS $$
DECLARE v_attempt agentrix.match_attempts%ROWTYPE;
BEGIN
    PERFORM agentrix.seal_evaluation_batch(
        '00000000-0000-0000-0000-000000000901', decode(repeat('41',32),'hex'), now());
    PERFORM agentrix.seal_match('00000000-0000-0000-0000-000000000921', now());
    PERFORM agentrix.enqueue_match(
        '00000000-0000-0000-0000-000000000941',
        '00000000-0000-0000-0000-000000000921', 10, now());
    SELECT * INTO v_attempt FROM agentrix.claim_match_job(
        '00000000-0000-0000-0000-000000000942', 'fixture-worker');
    PERFORM agentrix.record_match_seat_result(v_attempt.id, v_attempt.fencing_token,
        '00000000-0000-0000-0000-000000000931', 'completed', 60, 1, NULL);
    PERFORM agentrix.record_match_seat_result(v_attempt.id, v_attempt.fencing_token,
        '00000000-0000-0000-0000-000000000932', 'completed', 50, 2, NULL);
    PERFORM agentrix.record_match_replay(v_attempt.id, v_attempt.fencing_token,
        '00000000-0000-0000-0000-000000000209', 'agentrix-replay', '1', 1000);
    PERFORM agentrix.verify_match_replay(v_attempt.id, now());
    PERFORM agentrix.accept_match_attempt(v_attempt.id, v_attempt.fencing_token,
        decode(repeat('02',32),'hex'), decode(repeat('03',32),'hex'), '1',
        decode(repeat('42',32),'hex'));
    RETURN v_attempt.id;
END;
$$;

CREATE FUNCTION test_support.add_scored_fixture()
RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    PERFORM test_support.complete_fixture_match();
    INSERT INTO agentrix.judgements(
        id, contest_id, contest_task_id, submission_id, batch_id, build_attempt_id,
        evaluation_policy_version_id, scoring_policy_version_id, score_scope,
        state, verdict, engine_score, task_points, penalty_seconds, wins, draws,
        losses, effective, judged_at
    ) VALUES
    ('00000000-0000-0000-0000-000000000951', '00000000-0000-0000-0000-000000000701',
     '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000801',
     '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000811',
     '00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000402',
     'official', 'complete', 'accepted', 60, 60, 0, 1, 0, 0, false, now()),
    ('00000000-0000-0000-0000-000000000952', '00000000-0000-0000-0000-000000000701',
     '00000000-0000-0000-0000-000000000731', '00000000-0000-0000-0000-000000000802',
     '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000812',
     '00000000-0000-0000-0000-000000000401', '00000000-0000-0000-0000-000000000402',
     'official', 'complete', 'accepted', 50, 50, 0, 0, 0, 1, false, now())
    ON CONFLICT (id) DO NOTHING;
    INSERT INTO agentrix.judgement_matches(judgement_id, submission_id, batch_id, match_id, seat_id, test_case_id) VALUES
    ('00000000-0000-0000-0000-000000000951', '00000000-0000-0000-0000-000000000801', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000921', '00000000-0000-0000-0000-000000000931', '00000000-0000-0000-0000-000000000751'),
    ('00000000-0000-0000-0000-000000000952', '00000000-0000-0000-0000-000000000802', '00000000-0000-0000-0000-000000000901', '00000000-0000-0000-0000-000000000921', '00000000-0000-0000-0000-000000000932', '00000000-0000-0000-0000-000000000751')
    ON CONFLICT DO NOTHING;
    PERFORM agentrix.make_judgement_effective(
        '00000000-0000-0000-0000-000000000951', now(), 'fixture-effective-a');
    PERFORM agentrix.make_judgement_effective(
        '00000000-0000-0000-0000-000000000952', now(), 'fixture-effective-b');
END;
$$;
