SET search_path = agentrix, pg_catalog;

CREATE FUNCTION recalculate_score_cells(
    p_revision_id uuid, p_entry_id uuid DEFAULT NULL, p_task_id uuid DEFAULT NULL
) RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_revision score_revisions%ROWTYPE;
BEGIN
    SELECT * INTO v_revision FROM score_revisions WHERE id = p_revision_id FOR UPDATE;
    IF NOT FOUND OR v_revision.state <> 'building' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'score revision is not building';
    END IF;
    DELETE FROM score_cells
     WHERE revision_id = p_revision_id
       AND (p_entry_id IS NULL OR contest_entry_id = p_entry_id)
       AND (p_task_id IS NULL OR contest_task_id = p_task_id);

    INSERT INTO score_cells(
        revision_id, contest_id, contest_entry_id, contest_task_id,
        selected_submission_id, selected_judgement_id, task_points, solved,
        penalty_seconds, wins, draws, losses, computed_at
    )
    WITH base AS (
        SELECT s.contest_entry_id, s.contest_task_id, s.id AS submission_id,
               s.submitted_at, j.id AS judgement_id, j.judged_at, j.verdict,
               j.wins, j.draws, j.losses, sp.kind, sp.parameters,
               sp.disqualification_task_points, ct.max_task_points,
               CASE
                 WHEN j.verdict = 'disqualified' THEN sp.disqualification_task_points
                 WHEN j.verdict IN ('wrong_answer', 'compile_error', 'protocol_error', 'bot_error') THEN 0::numeric
                 WHEN sp.kind = 'aggregate_points' THEN COALESCE(
                     (SELECT sum(jc.task_points * CASE
                                 WHEN sp.parameters->>'group_aggregation' = 'weighted_sum'
                                 THEN tc.weight * tg.weight ELSE 1 END)
                        FROM judgement_cases jc
                        JOIN test_cases tc ON tc.id = jc.test_case_id
                        JOIN test_groups tg ON tg.id = tc.group_id
                       WHERE jc.judgement_id = j.id),
                     j.task_points, 0)
                 WHEN sp.kind = 'league_points' THEN
                     j.wins * (sp.parameters->>'win_points')::numeric
                     + j.draws * (sp.parameters->>'draw_points')::numeric
                     + j.losses * (sp.parameters->>'loss_points')::numeric
                 WHEN sp.kind = 'icpc_pass_fail' AND j.verdict = 'accepted' THEN ct.max_task_points
                 WHEN sp.kind = 'icpc_pass_fail' THEN 0::numeric
                 ELSE COALESCE(j.task_points, 0)
               END::numeric(24,8) AS computed_points,
               CASE WHEN sp.kind = 'icpc_pass_fail' AND j.verdict = 'accepted' THEN
                    GREATEST(0, extract(epoch FROM (s.submitted_at - c.starts_at)))::bigint
                    + (sp.parameters->>'wrong_submission_penalty_minutes')::bigint * 60 * (
                        SELECT count(*) FROM submissions sw
                        JOIN judgements jw ON jw.submission_id = sw.id
                         AND jw.effective AND jw.score_scope = 'official'
                       WHERE sw.contest_entry_id = s.contest_entry_id
                         AND sw.contest_task_id = s.contest_task_id
                         AND sw.submitted_at < s.submitted_at
                         AND jw.verdict NOT IN ('accepted', 'infrastructure_error')
                      )
                    ELSE j.penalty_seconds END AS computed_penalty
          FROM submissions s
          JOIN contests c ON c.id = s.contest_id
          JOIN contest_tasks ct ON ct.id = s.contest_task_id
          JOIN judgements j ON j.submission_id = s.id AND j.effective AND j.score_scope = 'official'
          JOIN scoring_policy_versions sp ON sp.id = j.scoring_policy_version_id
          JOIN evaluation_batches eb ON eb.id = j.batch_id AND eb.purpose IN ('official', 'rejudge')
         WHERE s.contest_id = v_revision.contest_id AND s.disposition = 'eligible'
           AND (v_revision.cutoff_submitted_at IS NULL OR s.submitted_at <= v_revision.cutoff_submitted_at)
           AND (p_entry_id IS NULL OR s.contest_entry_id = p_entry_id)
           AND (p_task_id IS NULL OR s.contest_task_id = p_task_id)
           AND j.verdict <> 'infrastructure_error'
    ), candidates AS (
        SELECT base.*,
               row_number() OVER (
                   PARTITION BY contest_entry_id, contest_task_id
                   ORDER BY
                     CASE WHEN kind = 'icpc_pass_fail' AND verdict = 'accepted' THEN 0
                          WHEN kind = 'icpc_pass_fail' THEN 1 ELSE 0 END,
                     CASE WHEN kind = 'icpc_pass_fail' AND verdict = 'accepted' THEN submitted_at END ASC,
                     CASE WHEN kind <> 'icpc_pass_fail' THEN computed_points END DESC NULLS LAST,
                     judged_at ASC, submission_id ASC
               ) AS choice
          FROM base
    )
    SELECT p_revision_id, v_revision.contest_id, contest_entry_id, contest_task_id,
           submission_id, judgement_id,
           computed_points, verdict = 'accepted', computed_penalty,
           wins, draws, losses, clock_timestamp()
      FROM candidates WHERE choice = 1;
END;
$$;

CREATE FUNCTION recalculate_score_rows(p_revision_id uuid)
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_revision score_revisions%ROWTYPE;
BEGIN
    SELECT * INTO v_revision FROM score_revisions WHERE id = p_revision_id FOR UPDATE;
    IF NOT FOUND OR v_revision.state <> 'building' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'score revision is not building';
    END IF;
    DELETE FROM score_rows WHERE revision_id = p_revision_id;
    INSERT INTO score_rows(
        revision_id, contest_id, division_id, contest_entry_id, rank,
        total_points, solved_count, total_penalty_seconds, total_wins,
        tie_break_value, computed_at
    )
    WITH totals AS (
        SELECT ce.division_id, ce.id AS contest_entry_id,
               COALESCE(sum(sc.task_points), 0)::numeric(24,8) AS total_points,
               count(*) FILTER (WHERE sc.solved)::integer AS solved_count,
               COALESCE(sum(sc.penalty_seconds), 0)::bigint AS total_penalty_seconds,
               COALESCE(sum(sc.wins), 0)::integer AS total_wins
          FROM contest_entries ce
          JOIN score_cells sc ON sc.revision_id = p_revision_id AND sc.contest_entry_id = ce.id
         WHERE ce.contest_id = v_revision.contest_id
           AND ce.status = 'accepted' AND ce.eligibility = 'eligible'
         GROUP BY ce.division_id, ce.id
    ), ranked AS (
        SELECT t.*,
               CASE WHEN c.rank_method = 'rank'
                    THEN rank() OVER (PARTITION BY t.division_id ORDER BY
                        t.total_points DESC, t.solved_count DESC,
                        t.total_penalty_seconds ASC, t.total_wins DESC)
                    ELSE dense_rank() OVER (PARTITION BY t.division_id ORDER BY
                        t.total_points DESC, t.solved_count DESC,
                        t.total_penalty_seconds ASC, t.total_wins DESC)
               END::integer AS computed_rank
          FROM totals t JOIN contests c ON c.id = v_revision.contest_id
    )
    SELECT p_revision_id, v_revision.contest_id, division_id, contest_entry_id,
           computed_rank, total_points, solved_count, total_penalty_seconds,
           total_wins, total_wins::numeric(24,8), clock_timestamp()
      FROM ranked;
END;
$$;

CREATE FUNCTION rebuild_scoreboard(
    p_revision_id uuid, p_contest_id uuid, p_cutoff_submitted_at timestamptz,
    p_created_at timestamptz
) RETURNS score_revisions
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_revision score_revisions%ROWTYPE; v_number bigint; v_event bigint;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(p_contest_id::text, 0));
    SELECT * INTO v_revision FROM score_revisions WHERE id = p_revision_id;
    IF FOUND THEN RETURN v_revision; END IF;
    PERFORM 1 FROM contests WHERE id = p_contest_id FOR SHARE;
    IF NOT FOUND THEN RAISE EXCEPTION USING ERRCODE = 'P0002', MESSAGE = 'contest not found'; END IF;
    SELECT COALESCE(max(revision_number), 0) + 1 INTO v_number
      FROM score_revisions WHERE contest_id = p_contest_id;
    SELECT COALESCE(max(id), 0) INTO v_event FROM score_events WHERE contest_id = p_contest_id;
    INSERT INTO score_revisions(id, contest_id, revision_number, state, source_kind,
                                cutoff_submitted_at, source_event_through, created_at)
    VALUES (p_revision_id, p_contest_id, v_number, 'building', 'full',
            p_cutoff_submitted_at, v_event, p_created_at);
    PERFORM recalculate_score_cells(p_revision_id);
    PERFORM recalculate_score_rows(p_revision_id);
    UPDATE score_revisions SET state = 'complete', completed_at = clock_timestamp()
     WHERE id = p_revision_id RETURNING * INTO v_revision;
    IF p_cutoff_submitted_at IS NULL THEN
        UPDATE score_events SET applied_revision_id = p_revision_id
         WHERE contest_id = p_contest_id AND id <= v_event AND applied_revision_id IS NULL;
    END IF;
    RETURN v_revision;
END;
$$;

CREATE FUNCTION rebuild_scoreboard_incremental(
    p_revision_id uuid, p_contest_id uuid, p_created_at timestamptz
) RETURNS score_revisions
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_revision score_revisions%ROWTYPE; v_previous score_revisions%ROWTYPE;
        v_number bigint; v_event bigint; v_change record;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(p_contest_id::text, 0));
    SELECT * INTO v_revision FROM score_revisions WHERE id = p_revision_id;
    IF FOUND THEN RETURN v_revision; END IF;
    SELECT * INTO v_previous FROM score_revisions
     WHERE contest_id = p_contest_id AND state = 'complete' AND cutoff_submitted_at IS NULL
     ORDER BY revision_number DESC LIMIT 1;
    IF NOT FOUND THEN
        RETURN rebuild_scoreboard(p_revision_id, p_contest_id, NULL, p_created_at);
    END IF;
    v_number := v_previous.revision_number + 1;
    SELECT COALESCE(max(id), v_previous.source_event_through) INTO v_event
      FROM score_events WHERE contest_id = p_contest_id;
    INSERT INTO score_revisions(id, contest_id, revision_number, state, source_kind,
                                cutoff_submitted_at, source_event_through, created_at)
    VALUES (p_revision_id, p_contest_id, v_number, 'building', 'incremental',
            NULL, v_event, p_created_at);
    INSERT INTO score_cells
    SELECT p_revision_id, contest_id, contest_entry_id, contest_task_id,
           selected_submission_id, selected_judgement_id, task_points, solved,
           penalty_seconds, wins, draws, losses, p_created_at
      FROM score_cells WHERE revision_id = v_previous.id;

    IF EXISTS (
        SELECT 1 FROM score_events WHERE contest_id = p_contest_id
         AND id > v_previous.source_event_through
         AND (contest_entry_id IS NULL OR contest_task_id IS NULL)
    ) THEN
        PERFORM recalculate_score_cells(p_revision_id);
    ELSE
        FOR v_change IN
            SELECT DISTINCT contest_entry_id, contest_task_id FROM score_events
             WHERE contest_id = p_contest_id AND id > v_previous.source_event_through
               AND id <= v_event
        LOOP
            PERFORM recalculate_score_cells(p_revision_id, v_change.contest_entry_id, v_change.contest_task_id);
        END LOOP;
    END IF;
    PERFORM recalculate_score_rows(p_revision_id);
    UPDATE score_revisions SET state = 'complete', completed_at = clock_timestamp()
     WHERE id = p_revision_id RETURNING * INTO v_revision;
    UPDATE score_events SET applied_revision_id = p_revision_id
     WHERE contest_id = p_contest_id AND id <= v_event AND applied_revision_id IS NULL;
    RETURN v_revision;
END;
$$;

CREATE FUNCTION publish_scoreboard(
    p_publication_id uuid, p_contest_id uuid, p_source_revision_id uuid,
    p_audience text, p_kind text, p_cutoff_submitted_at timestamptz,
    p_published_by_user_id uuid, p_published_at timestamptz
) RETURNS scoreboard_publications
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
DECLARE v_publication scoreboard_publications%ROWTYPE; v_revision score_revisions%ROWTYPE;
        v_number bigint;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(p_contest_id::text || ':' || p_audience, 0));
    SELECT * INTO v_publication FROM scoreboard_publications WHERE id = p_publication_id;
    IF FOUND THEN RETURN v_publication; END IF;
    SELECT * INTO v_revision FROM score_revisions
     WHERE id = p_source_revision_id AND contest_id = p_contest_id FOR SHARE;
    IF NOT FOUND OR v_revision.state <> 'complete' THEN
        RAISE EXCEPTION USING ERRCODE = '55000', MESSAGE = 'source score revision is not complete';
    END IF;
    IF p_kind = 'freeze' AND v_revision.cutoff_submitted_at IS DISTINCT FROM p_cutoff_submitted_at THEN
        RAISE EXCEPTION USING ERRCODE = '23514', MESSAGE = 'freeze publication cutoff must equal revision submission cutoff';
    END IF;
    SELECT COALESCE(max(publication_number), 0) + 1 INTO v_number
      FROM scoreboard_publications WHERE contest_id = p_contest_id AND audience = p_audience;
    INSERT INTO scoreboard_publications(
        id, contest_id, source_revision_id, publication_number, audience,
        kind, cutoff_submitted_at, published_by_user_id, published_at
    ) VALUES (
        p_publication_id, p_contest_id, p_source_revision_id, v_number, p_audience,
        p_kind, p_cutoff_submitted_at, p_published_by_user_id, p_published_at
    ) RETURNING * INTO v_publication;
    INSERT INTO scoreboard_publication_rows
    SELECT p_publication_id, contest_id, division_id, contest_entry_id, rank,
           total_points, solved_count, total_penalty_seconds, total_wins, tie_break_value
      FROM score_rows WHERE revision_id = p_source_revision_id;
    INSERT INTO scoreboard_publication_cells
    SELECT p_publication_id, contest_id, contest_entry_id, contest_task_id,
           task_points, solved, penalty_seconds
      FROM score_cells WHERE revision_id = p_source_revision_id;
    INSERT INTO contest_events(contest_id, event_key, event_kind, audience, entity_id, payload, occurred_at)
    VALUES (p_contest_id, 'scoreboard:' || p_publication_id, 'scoreboard_published',
            p_audience, p_publication_id,
            jsonb_build_object('publication_id', p_publication_id, 'kind', p_kind), p_published_at);
    RETURN v_publication;
END;
$$;
