BEGIN;
SET search_path = agentrix, pg_catalog;
SET LOCAL enable_seqscan = off;

SELECT test_support.assert_true(position('matches_batch_state' IN test_support.explain_text(
 $query$SELECT * FROM agentrix.matches WHERE batch_id='00000000-0000-0000-0000-000000000901' AND state='planned'$query$)) > 0,
 'batch/state match query uses its measured index path');
SELECT test_support.assert_true(position('match_jobs_claim' IN test_support.explain_text(
 $query$SELECT id FROM agentrix.match_jobs WHERE state='available' ORDER BY priority DESC,available_at,id LIMIT 1$query$)) > 0
 OR NOT EXISTS (SELECT 1 FROM match_jobs), 'claim query has its partial index path');
SELECT test_support.expect_error($sql$
 DELETE FROM submissions WHERE id='00000000-0000-0000-0000-000000000801'
$sql$, '55000', 'immutable');
SELECT test_support.expect_error($sql$
 DELETE FROM game_releases WHERE id='00000000-0000-0000-0000-000000000302'
$sql$, '55000', 'immutable');
SELECT test_support.assert_true(NOT EXISTS (
 SELECT 1 FROM pg_namespace n,
 LATERAL aclexplode(COALESCE(n.nspacl, acldefault('n',n.nspowner))) a
 WHERE n.nspname='agentrix' AND a.grantee=0 AND a.privilege_type='USAGE'
), 'PUBLIC has no schema usage');
SELECT test_support.assert_true((SELECT count(*)=0 FROM information_schema.role_table_grants
 WHERE grantee='PUBLIC' AND table_schema='agentrix'), 'PUBLIC has no table grants');
SELECT test_support.assert_true((SELECT count(*) = 3 FROM pg_indexes WHERE schemaname='agentrix'
 AND indexname IN ('submissions_score_candidates','judgements_one_effective_official','score_rows_ranking')),
 'ranking and effective-judgement indexes exist');

ROLLBACK;
