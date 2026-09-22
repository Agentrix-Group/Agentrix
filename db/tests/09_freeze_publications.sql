BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT test_support.add_scored_fixture();
SELECT rebuild_scoreboard('80000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000701',now()-interval '85 minutes',now());
SELECT test_support.assert_true((SELECT count(*)=1 FROM score_cells WHERE revision_id='80000000-0000-0000-0000-000000000001'), 'freeze uses submission time: late-judged earlier submission is visible, later submission is hidden');
SELECT publish_scoreboard('80000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000701','80000000-0000-0000-0000-000000000001','public','freeze',now()-interval '85 minutes','00000000-0000-0000-0000-000000000003',now());

INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,evaluation_policy_version_id,
 scoring_policy_version_id,score_scope,state,verdict,task_points,effective,judged_at) VALUES
('80000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000811','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000406','official','complete','accepted',80,false,now());
INSERT INTO judgement_matches(judgement_id,submission_id,batch_id,match_id,seat_id,test_case_id) VALUES
('80000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000921','00000000-0000-0000-0000-000000000931','00000000-0000-0000-0000-000000000751');
INSERT INTO rejudge_batches(id,contest_id,scope_kind,scope_id,reason,state,requested_by_user_id,reviewed_by_user_id,idempotency_key,created_at,reviewed_at) VALUES
('80000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000701','submission','00000000-0000-0000-0000-000000000801','freeze rejudge','reviewed','00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000003','freeze-rejudge',now(),now());
INSERT INTO rejudge_items(rejudge_batch_id,submission_id,previous_judgement_id,candidate_judgement_id,state,scoring_policy_before_id,scoring_policy_after_id) VALUES
('80000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000951','80000000-0000-0000-0000-000000000003','approved','00000000-0000-0000-0000-000000000402','00000000-0000-0000-0000-000000000406');
SELECT apply_rejudge_batch('80000000-0000-0000-0000-000000000004',now());
SELECT rebuild_scoreboard('80000000-0000-0000-0000-000000000005','00000000-0000-0000-0000-000000000701',NULL,now());
SELECT test_support.assert_true((SELECT task_points=60 FROM scoreboard_publication_cells WHERE publication_id='80000000-0000-0000-0000-000000000002'), 'rejudge during freeze cannot mutate historical public snapshot');
SELECT test_support.assert_true((SELECT task_points=80 FROM score_cells WHERE revision_id='80000000-0000-0000-0000-000000000005' AND contest_entry_id='00000000-0000-0000-0000-000000000721'), 'jury current revision sees rejudge immediately');
SELECT publish_scoreboard('80000000-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000701','80000000-0000-0000-0000-000000000005','public','unfreeze',NULL,'00000000-0000-0000-0000-000000000003',now());
SELECT test_support.assert_true((SELECT task_points=80 FROM scoreboard_publication_cells WHERE publication_id='80000000-0000-0000-0000-000000000006' AND contest_entry_id='00000000-0000-0000-0000-000000000721'), 'unfreeze publishes a new complete revision');
SELECT test_support.expect_error($sql$
 UPDATE scoreboard_publication_cells SET task_points=999 WHERE publication_id='80000000-0000-0000-0000-000000000002'
$sql$, '55000', 'immutable');

ROLLBACK;

