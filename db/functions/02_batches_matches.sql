SET search_path = agentrix, pg_catalog;

CREATE FUNCTION seal_evaluation_batch(
    p_batch_id uuid, p_roster_digest sha256_digest, p_sealed_at timestamptz
) RETURNS evaluation_batches
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_batch evaluation_batches%ROWTYPE;
BEGIN
    SELECT * INTO v_batch FROM evaluation_batches WHERE id = p_batch_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'batch not found'; END IF;
    IF v_batch.state <> 'draft' THEN
        IF v_batch.roster_digest = p_roster_digest THEN RETURN v_batch; END IF;
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'batch is already sealed';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM batch_roster WHERE batch_id = p_batch_id) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'batch roster is empty';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM batch_roster br
          LEFT JOIN artifacts a ON a.id = br.executable_artifact_id
          LEFT JOIN build_attempts ba ON ba.id = br.build_attempt_id
          LEFT JOIN baseline_programs bp ON bp.id = br.baseline_program_id
         WHERE br.batch_id = p_batch_id
           AND (
             a.publication_state IS DISTINCT FROM 'ready'
             OR a.sha256 IS DISTINCT FROM br.executable_digest
             OR (br.source_kind = 'submission' AND (
                    ba.state IS DISTINCT FROM 'succeeded'
                    OR ba.executable_artifact_id IS DISTINCT FROM br.executable_artifact_id
                    OR NOT EXISTS (SELECT 1 FROM submissions s WHERE s.id = br.submission_id AND s.disposition = 'eligible')
                ))
             OR (br.source_kind = 'baseline' AND (
                    bp.game_release_id IS DISTINCT FROM v_batch.game_release_id
                    OR bp.executable_artifact_id IS DISTINCT FROM br.executable_artifact_id
                    OR bp.executable_digest IS DISTINCT FROM br.executable_digest
                ))
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'roster contains an unready or incompatible program';
    END IF;
    UPDATE evaluation_batches
       SET state = 'sealed', roster_digest = p_roster_digest, sealed_at = p_sealed_at
     WHERE id = p_batch_id RETURNING * INTO v_batch;
    RETURN v_batch;
END;
$$;

CREATE FUNCTION seal_match(p_match_id uuid, p_sealed_at timestamptz)
RETURNS matches
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_match matches%ROWTYPE; v_batch evaluation_batches%ROWTYPE;
        v_min smallint; v_max smallint; v_count integer;
BEGIN
    SELECT * INTO v_match FROM matches WHERE id = p_match_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'match not found'; END IF;
    IF v_match.state <> 'planned' THEN RETURN v_match; END IF;
    SELECT * INTO v_batch FROM evaluation_batches WHERE id = v_match.batch_id FOR UPDATE;
    IF v_batch.state NOT IN ('sealed', 'running') THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'batch must be sealed before matches';
    END IF;
    IF (v_match.purpose = 'official' AND v_batch.purpose <> 'official')
       OR (v_match.purpose = 'rejudge' AND v_batch.purpose <> 'rejudge')
       OR (v_match.purpose = 'practice' AND v_batch.purpose <> 'practice') THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'match purpose is incompatible with batch purpose';
    END IF;
    SELECT min_players, max_players INTO v_min, v_max
      FROM game_releases WHERE id = v_match.game_release_id;
    SELECT count(*) INTO v_count FROM match_seats WHERE match_id = p_match_id;
    IF v_count < v_min OR v_count > v_max THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = format('seat count %s is outside [%s,%s]', v_count, v_min, v_max);
    END IF;
    IF NOT v_match.allow_duplicate_programs AND EXISTS (
        SELECT 1 FROM match_seats WHERE match_id = p_match_id
         GROUP BY roster_item_id HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'duplicate program seats are disabled for this match';
    END IF;
    UPDATE matches SET state = 'sealed', sealed_at = p_sealed_at
     WHERE id = p_match_id RETURNING * INTO v_match;
    UPDATE evaluation_batches SET state = 'running'
     WHERE id = v_match.batch_id AND state = 'sealed';
    RETURN v_match;
END;
$$;

CREATE FUNCTION enqueue_match(
    p_job_id uuid, p_match_id uuid, p_priority integer,
    p_available_at timestamptz
) RETURNS match_jobs
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_match matches%ROWTYPE; v_job match_jobs%ROWTYPE; v_max smallint;
BEGIN
    SELECT * INTO v_job FROM match_jobs WHERE match_id = p_match_id;
    IF FOUND THEN RETURN v_job; END IF;
    SELECT * INTO v_match FROM matches WHERE id = p_match_id FOR UPDATE;
    IF NOT FOUND OR v_match.state <> 'sealed' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'only a sealed match can be enqueued';
    END IF;
    SELECT ep.max_match_attempts INTO v_max
      FROM evaluation_batches eb
      JOIN evaluation_policy_versions ep ON ep.id = eb.evaluation_policy_version_id
     WHERE eb.id = v_match.batch_id;
    INSERT INTO match_jobs(id, match_id, state, priority, available_at, max_attempts,
                           created_at, updated_at)
    VALUES (p_job_id, p_match_id, 'available', p_priority, p_available_at, v_max,
            clock_timestamp(), clock_timestamp())
    RETURNING * INTO v_job;
    UPDATE matches SET state = 'queued' WHERE id = p_match_id;
    RETURN v_job;
END;
$$;

CREATE FUNCTION claim_match_job(
    p_attempt_id uuid, p_worker_id text
) RETURNS match_attempts
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_job match_jobs%ROWTYPE; v_attempt match_attempts%ROWTYPE;
        v_lease_seconds integer; v_spec sha256_digest;
        p_now timestamptz := clock_timestamp();
BEGIN
    SELECT * INTO v_attempt FROM match_attempts WHERE id = p_attempt_id;
    IF FOUND THEN RETURN v_attempt; END IF;

    UPDATE match_attempts ma
       SET state = 'rejected', failure_origin = 'worker_infrastructure',
           failure_code = 'lease_expired_retry_limit', ended_at = p_now
      FROM match_jobs mj
     WHERE mj.id = ma.job_id AND mj.state = 'leased'
       AND mj.lease_expires_at <= p_now AND mj.attempt_count >= mj.max_attempts
       AND ma.fencing_token = mj.fencing_token AND ma.state = 'running';
    UPDATE matches m SET state = 'failed'
      FROM match_jobs mj
     WHERE mj.match_id = m.id AND mj.state = 'leased'
       AND mj.lease_expires_at <= p_now AND mj.attempt_count >= mj.max_attempts;
    UPDATE match_jobs
       SET state = 'dead', lease_owner = NULL, lease_expires_at = NULL,
           last_error_kind = 'worker_infrastructure', updated_at = p_now
     WHERE state = 'leased' AND lease_expires_at <= p_now AND attempt_count >= max_attempts;

    SELECT * INTO v_job
      FROM match_jobs
     WHERE ((state = 'available' AND available_at <= p_now)
            OR (state = 'leased' AND lease_expires_at <= p_now))
       AND attempt_count < max_attempts
     ORDER BY priority DESC, available_at, id
     FOR UPDATE SKIP LOCKED
     LIMIT 1;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'no claimable match job';
    END IF;

    IF v_job.state = 'leased' THEN
        UPDATE match_attempts
           SET state = 'rejected', failure_origin = 'worker_infrastructure',
               failure_code = 'lease_expired', ended_at = p_now
         WHERE job_id = v_job.id AND fencing_token = v_job.fencing_token AND state = 'running';
    END IF;
    SELECT ep.lease_duration_seconds, m.specification_digest
      INTO v_lease_seconds, v_spec
      FROM matches m
      JOIN evaluation_batches eb ON eb.id = m.batch_id
      JOIN evaluation_policy_versions ep ON ep.id = eb.evaluation_policy_version_id
     WHERE m.id = v_job.match_id;
    UPDATE match_jobs
       SET state = 'leased', attempt_count = attempt_count + 1,
           fencing_token = fencing_token + 1, lease_owner = p_worker_id,
           lease_expires_at = p_now + make_interval(secs => v_lease_seconds),
           last_heartbeat_at = p_now, updated_at = p_now
     WHERE id = v_job.id RETURNING * INTO v_job;
    INSERT INTO match_attempts(
        id, match_id, job_id, attempt_number, worker_id, fencing_token, state,
        scheduled_specification_digest, started_at, heartbeat_at
    ) VALUES (
        p_attempt_id, v_job.match_id, v_job.id, v_job.attempt_count, p_worker_id,
        v_job.fencing_token, 'running', v_spec, p_now, p_now
    ) RETURNING * INTO v_attempt;
    UPDATE matches SET state = 'running' WHERE id = v_job.match_id AND state = 'queued';
    RETURN v_attempt;
END;
$$;

CREATE FUNCTION heartbeat_match_attempt(
    p_attempt_id uuid, p_fencing_token bigint
) RETURNS match_attempts
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_attempt match_attempts%ROWTYPE; v_seconds integer;
        p_now timestamptz := clock_timestamp();
BEGIN
    SELECT ma.* INTO v_attempt
      FROM match_attempts ma JOIN match_jobs mj ON mj.id = ma.job_id
     WHERE ma.id = p_attempt_id AND ma.fencing_token = p_fencing_token
       AND ma.state = 'running' AND mj.state = 'leased'
       AND mj.fencing_token = p_fencing_token AND mj.lease_expires_at > p_now
     FOR UPDATE OF ma, mj;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'stale or expired fencing token'; END IF;
    SELECT ep.lease_duration_seconds INTO v_seconds
      FROM matches m JOIN evaluation_batches eb ON eb.id = m.batch_id
      JOIN evaluation_policy_versions ep ON ep.id = eb.evaluation_policy_version_id
     WHERE m.id = v_attempt.match_id;
    UPDATE match_attempts SET heartbeat_at = p_now WHERE id = p_attempt_id RETURNING * INTO v_attempt;
    UPDATE match_jobs SET last_heartbeat_at = p_now,
           lease_expires_at = p_now + make_interval(secs => v_seconds), updated_at = p_now
     WHERE id = v_attempt.job_id;
    RETURN v_attempt;
END;
$$;

CREATE FUNCTION record_match_seat_result(
    p_attempt_id uuid, p_fencing_token bigint, p_seat_id uuid,
    p_outcome text, p_engine_score numeric, p_placement smallint,
    p_metrics_artifact_id uuid
) RETURNS match_seat_results
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_attempt match_attempts%ROWTYPE; v_result match_seat_results%ROWTYPE;
BEGIN
    SELECT ma.* INTO v_attempt
      FROM match_attempts ma JOIN match_jobs mj ON mj.id = ma.job_id
     WHERE ma.id = p_attempt_id AND ma.fencing_token = p_fencing_token
       AND ma.state = 'running' AND mj.state = 'leased'
       AND mj.fencing_token = p_fencing_token AND mj.lease_expires_at > clock_timestamp()
     FOR SHARE OF ma, mj;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'stale or expired fencing token'; END IF;
    SELECT * INTO v_result FROM match_seat_results
     WHERE attempt_id = p_attempt_id AND seat_id = p_seat_id;
    IF FOUND THEN
        IF (v_result.outcome, v_result.engine_score, v_result.placement, v_result.metrics_artifact_id)
           IS DISTINCT FROM (p_outcome, p_engine_score, p_placement, p_metrics_artifact_id) THEN
            RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'seat result already recorded with different values';
        END IF;
        RETURN v_result;
    END IF;
    INSERT INTO match_seat_results(match_id, attempt_id, seat_id, outcome,
                                   engine_score, placement, metrics_artifact_id, created_at)
    VALUES (v_attempt.match_id, p_attempt_id, p_seat_id, p_outcome,
            p_engine_score, p_placement, p_metrics_artifact_id, clock_timestamp())
    RETURNING * INTO v_result;
    RETURN v_result;
END;
$$;

CREATE FUNCTION record_match_replay(
    p_attempt_id uuid, p_fencing_token bigint, p_artifact_id uuid,
    p_format text, p_format_version text, p_duration_ticks bigint
) RETURNS match_replays
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_attempt match_attempts%ROWTYPE; v_replay match_replays%ROWTYPE;
BEGIN
    SELECT ma.* INTO v_attempt
      FROM match_attempts ma JOIN match_jobs mj ON mj.id = ma.job_id
     WHERE ma.id = p_attempt_id AND ma.fencing_token = p_fencing_token
       AND ma.state = 'running' AND mj.state = 'leased'
       AND mj.fencing_token = p_fencing_token AND mj.lease_expires_at > clock_timestamp()
     FOR SHARE OF ma, mj;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'stale or expired fencing token'; END IF;
    IF NOT EXISTS (SELECT 1 FROM artifacts WHERE id = p_artifact_id AND kind = 'replay' AND publication_state = 'ready') THEN
        RAISE EXCEPTION USING ERRCODE = '22023', MESSAGE = 'replay artifact is not ready';
    END IF;
    SELECT * INTO v_replay FROM match_replays WHERE attempt_id = p_attempt_id;
    IF FOUND THEN RETURN v_replay; END IF;
    INSERT INTO match_replays(match_id, attempt_id, artifact_id, format, format_version,
                              duration_ticks, state)
    VALUES (v_attempt.match_id, p_attempt_id, p_artifact_id, p_format, p_format_version,
            p_duration_ticks, 'pending')
    RETURNING * INTO v_replay;
    RETURN v_replay;
END;
$$;

CREATE FUNCTION verify_match_replay(p_attempt_id uuid, p_verified_at timestamptz)
RETURNS match_replays
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_replay match_replays%ROWTYPE;
BEGIN
    SELECT * INTO v_replay FROM match_replays WHERE attempt_id = p_attempt_id FOR UPDATE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'replay not found'; END IF;
    IF v_replay.state = 'verified' THEN RETURN v_replay; END IF;
    IF v_replay.state <> 'pending' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'replay cannot be verified from current state';
    END IF;
    UPDATE match_replays SET state = 'verified', verified_at = p_verified_at
     WHERE attempt_id = p_attempt_id RETURNING * INTO v_replay;
    RETURN v_replay;
END;
$$;

CREATE FUNCTION accept_match_attempt(
    p_attempt_id uuid, p_fencing_token bigint,
    p_engine_digest sha256_digest, p_rules_digest sha256_digest,
    p_protocol_version text, p_output_digest sha256_digest
) RETURNS matches
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_attempt match_attempts%ROWTYPE; v_match matches%ROWTYPE;
        v_release game_releases%ROWTYPE; v_requires_replay boolean;
        v_seats integer; v_results integer;
        p_ended_at timestamptz := clock_timestamp();
BEGIN
    SELECT m.* INTO v_match FROM matches m
      JOIN match_attempts ma ON ma.match_id = m.id WHERE ma.id = p_attempt_id FOR UPDATE OF m;
    IF v_match.accepted_attempt_id = p_attempt_id THEN RETURN v_match; END IF;
    IF v_match.accepted_attempt_id IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '23505', MESSAGE = 'match already has an accepted attempt';
    END IF;
    SELECT ma.* INTO v_attempt
      FROM match_attempts ma JOIN match_jobs mj ON mj.id = ma.job_id
     WHERE ma.id = p_attempt_id AND ma.fencing_token = p_fencing_token
       AND ma.state = 'running' AND mj.state = 'leased'
       AND mj.fencing_token = p_fencing_token AND mj.lease_expires_at > p_ended_at
     FOR UPDATE OF ma, mj;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'stale or expired fencing token'; END IF;
    SELECT * INTO v_release FROM game_releases WHERE id = v_match.game_release_id;
    IF v_attempt.scheduled_specification_digest <> v_match.specification_digest
       OR p_engine_digest <> v_release.engine_digest OR p_rules_digest <> v_release.rules_digest
       OR p_protocol_version <> v_release.protocol_version THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'worker output does not match scheduled engine, rules, protocol, or specification';
    END IF;
    SELECT count(*) INTO v_seats FROM match_seats WHERE match_id = v_match.id;
    SELECT count(*) INTO v_results FROM match_seat_results WHERE attempt_id = p_attempt_id;
    IF v_results <> v_seats THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'attempt does not contain exactly one result per seat';
    END IF;
    SELECT requires_verified_replay INTO v_requires_replay FROM contest_tasks WHERE id = v_match.contest_task_id;
    IF v_requires_replay AND NOT EXISTS (
        SELECT 1 FROM match_replays mr JOIN artifacts a ON a.id = mr.artifact_id
         WHERE mr.attempt_id = p_attempt_id AND mr.state IN ('verified', 'published')
           AND a.publication_state = 'ready'
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'verified replay is required';
    END IF;
    UPDATE match_attempts
       SET state = 'succeeded', ended_at = p_ended_at, output_digest = p_output_digest,
           observed_engine_digest = p_engine_digest, observed_rules_digest = p_rules_digest,
           observed_protocol_version = p_protocol_version
     WHERE id = p_attempt_id;
    UPDATE matches SET accepted_attempt_id = p_attempt_id, state = 'completed', completed_at = p_ended_at
     WHERE id = v_match.id RETURNING * INTO v_match;
    UPDATE match_jobs SET state = 'succeeded', lease_owner = NULL, lease_expires_at = NULL,
           updated_at = p_ended_at, last_error_kind = NULL
     WHERE id = v_attempt.job_id;
    RETURN v_match;
END;
$$;

CREATE FUNCTION fail_match_attempt(
    p_attempt_id uuid, p_fencing_token bigint,
    p_failure_origin text, p_failure_code text
) RETURNS match_jobs
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_attempt match_attempts%ROWTYPE; v_job match_jobs%ROWTYPE;
        p_ended_at timestamptz := clock_timestamp();
BEGIN
    SELECT ma.* INTO v_attempt FROM match_attempts ma JOIN match_jobs mj ON mj.id = ma.job_id
     WHERE ma.id = p_attempt_id AND ma.fencing_token = p_fencing_token
       AND ma.state = 'running' AND mj.state = 'leased' AND mj.fencing_token = p_fencing_token
       AND mj.lease_expires_at > p_ended_at
     FOR UPDATE OF ma, mj;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = '40001', MESSAGE = 'stale fencing token'; END IF;
    UPDATE match_attempts SET state = 'failed', failure_origin = p_failure_origin,
           failure_code = p_failure_code, ended_at = p_ended_at WHERE id = p_attempt_id;
    UPDATE match_jobs SET
        state = CASE WHEN attempt_count < max_attempts THEN 'available' ELSE 'dead' END,
        available_at = p_ended_at, lease_owner = NULL, lease_expires_at = NULL,
        last_error_kind = p_failure_origin, updated_at = p_ended_at
     WHERE id = v_attempt.job_id RETURNING * INTO v_job;
    UPDATE matches SET state = CASE WHEN v_job.state = 'dead' THEN 'failed' ELSE 'queued' END
     WHERE id = v_attempt.match_id;
    RETURN v_job;
END;
$$;
