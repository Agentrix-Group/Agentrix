-- Canonical Agentrix schema. This is a clean-install definition, not a migration.
CREATE SCHEMA agentrix;
SET search_path = agentrix, pg_catalog;

CREATE DOMAIN sha256_digest AS bytea
    CHECK (octet_length(VALUE) = 32);

CREATE DOMAIN nonnegative_score AS numeric(24, 8)
    CHECK (VALUE >= 0);

CREATE FUNCTION jsonb_has_only_keys(p_value jsonb, p_allowed text[])
RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
    SELECT COALESCE(bool_and(key = ANY(p_allowed)), true)
      FROM jsonb_object_keys(p_value) AS keys(key)
$$;

CREATE FUNCTION valid_evaluation_policy(p_kind text, p_parameters jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE AS $$
BEGIN
    IF jsonb_typeof(p_parameters) <> 'object' THEN RETURN false; END IF;
    CASE p_kind
      WHEN 'fixed_cases' THEN
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['suite_version'])
          AND jsonb_typeof(p_parameters->'suite_version') = 'string', false);
      WHEN 'reference_opponents' THEN
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['reference_roster_digest'])
          AND jsonb_typeof(p_parameters->'reference_roster_digest') = 'string', false);
      WHEN 'round_robin' THEN
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['rounds','allow_self_play'])
           AND jsonb_typeof(p_parameters->'rounds') = 'number'
           AND (p_parameters->>'rounds')::integer > 0
           AND jsonb_typeof(p_parameters->'allow_self_play') = 'boolean', false);
      WHEN 'league_schedule' THEN
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['schedule_version'])
          AND jsonb_typeof(p_parameters->'schedule_version') = 'string', false);
      ELSE RETURN false;
    END CASE;
EXCEPTION WHEN invalid_text_representation OR numeric_value_out_of_range THEN
    RETURN false;
END;
$$;

CREATE FUNCTION valid_scoring_policy(p_kind text, p_parameters jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE v numeric;
BEGIN
    IF jsonb_typeof(p_parameters) <> 'object' THEN RETURN false; END IF;
    CASE p_kind
      WHEN 'icpc_pass_fail' THEN
        v := (p_parameters->>'wrong_submission_penalty_minutes')::numeric;
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['wrong_submission_penalty_minutes','higher_is_better'])
          AND v >= 0 AND COALESCE((p_parameters->>'higher_is_better')::boolean, true), false);
      WHEN 'best_score' THEN
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['higher_is_better'])
          AND (p_parameters->>'higher_is_better')::boolean = true, false);
      WHEN 'aggregate_points' THEN
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['higher_is_better','group_aggregation'])
           AND (p_parameters->>'higher_is_better')::boolean = true
           AND p_parameters->>'group_aggregation' IN ('sum', 'weighted_sum'), false);
      WHEN 'league_points' THEN
        RETURN COALESCE(jsonb_has_only_keys(p_parameters, ARRAY['win_points','draw_points','loss_points'])
           AND jsonb_typeof(p_parameters->'win_points') = 'number'
           AND jsonb_typeof(p_parameters->'draw_points') = 'number'
           AND jsonb_typeof(p_parameters->'loss_points') = 'number'
           AND (p_parameters->>'win_points')::numeric >= (p_parameters->>'draw_points')::numeric
           AND (p_parameters->>'draw_points')::numeric >= (p_parameters->>'loss_points')::numeric, false);
      ELSE RETURN false;
    END CASE;
EXCEPTION WHEN invalid_text_representation OR numeric_value_out_of_range THEN
    RETURN false;
END;
$$;

CREATE FUNCTION reject_mutation_after_insert()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '55000',
        MESSAGE = format('%s rows are immutable', TG_TABLE_NAME);
END;
$$;

CREATE FUNCTION reject_mutation_when_sealed()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE v_state text;
BEGIN
    v_state := COALESCE(to_jsonb(OLD)->>'status', to_jsonb(OLD)->>'state');
    IF v_state IN ('sealed', 'running', 'complete', 'completed', 'applied', 'published')
       AND NEW IS DISTINCT FROM OLD THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = format('%s is immutable in state %s', TG_TABLE_NAME, v_state);
    END IF;
    IF TG_OP = 'DELETE' THEN RETURN OLD; END IF;
    RETURN NEW;
END;
$$;
