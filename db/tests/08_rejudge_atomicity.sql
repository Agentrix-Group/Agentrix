BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT test_support.add_scored_fixture();
INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,
 evaluation_policy_version_id,scoring_policy_version_id,score_scope,state,verdict,engine_score,
 task_points,effective,judged_at) VALUES
('70000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000811','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000406','official','complete','accepted',80,80,false,now());
INSERT INTO judgement_matches(judgement_id,submission_id,batch_id,match_id,seat_id,test_case_id) VALUES
('70000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000921','00000000-0000-0000-0000-000000000931','00000000-0000-0000-0000-000000000751');
INSERT INTO rejudge_batches(id,contest_id,scope_kind,scope_id,reason,state,requested_by_user_id,
 reviewed_by_user_id,idempotency_key,created_at,reviewed_at) VALUES
('70000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000701','submission','00000000-0000-0000-0000-000000000801','policy v2','reviewed','00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000003','rejudge-ok',now(),now());
INSERT INTO rejudge_items(rejudge_batch_id,submission_id,previous_judgement_id,candidate_judgement_id,state,
 scoring_policy_before_id,scoring_policy_after_id) VALUES
('70000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000951','70000000-0000-0000-0000-000000000001','approved','00000000-0000-0000-0000-000000000402','00000000-0000-0000-0000-000000000406');
SELECT test_support.assert_true((SELECT effective FROM judgements WHERE id='00000000-0000-0000-0000-000000000951')
 AND NOT (SELECT effective FROM judgements WHERE id='70000000-0000-0000-0000-000000000001'),
 'prepared and reviewed candidate does not affect the current judgement');
SELECT apply_rejudge_batch('70000000-0000-0000-0000-000000000010',now());
SELECT apply_rejudge_batch('70000000-0000-0000-0000-000000000010',now());
SELECT test_support.assert_true((SELECT NOT effective AND superseded_at IS NOT NULL FROM judgements WHERE id='00000000-0000-0000-0000-000000000951'), 'previous judgement history is preserved');
SELECT test_support.assert_true((SELECT effective FROM judgements WHERE id='70000000-0000-0000-0000-000000000001'), 'candidate is promoted exactly once');

INSERT INTO rejudge_batches(id,contest_id,scope_kind,scope_id,reason,state,requested_by_user_id,idempotency_key,created_at) VALUES
('70000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000701','task','00000000-0000-0000-0000-000000000731','cancel fixture','prepared','00000000-0000-0000-0000-000000000003','rejudge-cancel',now());
SELECT cancel_rejudge_batch('70000000-0000-0000-0000-000000000011',now());
SELECT test_support.assert_true((SELECT state='cancelled' FROM rejudge_batches WHERE id='70000000-0000-0000-0000-000000000011'), 'cancel leaves effective judgement unchanged');

-- A two-item apply fails on the second candidate; the first replacement must roll back.
INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,
 evaluation_policy_version_id,scoring_policy_version_id,score_scope,state,verdict,task_points,effective,judged_at) VALUES
('70000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000811','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000406','official','complete','accepted',90,false,now()),
('70000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','00000000-0000-0000-0000-000000000802','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000812','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000406','official','pending','accepted',70,false,now());
INSERT INTO judgement_matches(judgement_id,submission_id,batch_id,match_id,seat_id,test_case_id) VALUES
('70000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000801','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000921','00000000-0000-0000-0000-000000000931','00000000-0000-0000-0000-000000000751');
INSERT INTO rejudge_batches(id,contest_id,scope_kind,scope_id,reason,state,requested_by_user_id,
 reviewed_by_user_id,idempotency_key,created_at,reviewed_at) VALUES
('70000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000701','contest','00000000-0000-0000-0000-000000000701','rollback fixture','reviewed','00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000003','rejudge-rollback',now(),now());
INSERT INTO rejudge_items(rejudge_batch_id,submission_id,previous_judgement_id,candidate_judgement_id,state,scoring_policy_before_id,scoring_policy_after_id) VALUES
('70000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000801','70000000-0000-0000-0000-000000000001','70000000-0000-0000-0000-000000000002','approved','00000000-0000-0000-0000-000000000406','00000000-0000-0000-0000-000000000406'),
('70000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000802','00000000-0000-0000-0000-000000000952','70000000-0000-0000-0000-000000000003','approved','00000000-0000-0000-0000-000000000402','00000000-0000-0000-0000-000000000406');
SELECT test_support.expect_error($sql$
 SELECT apply_rejudge_batch('70000000-0000-0000-0000-000000000012',now())
$sql$, '23514', 'invalid rejudge candidate');
SELECT test_support.assert_true((SELECT effective FROM judgements WHERE id='70000000-0000-0000-0000-000000000001'), 'mid-apply failure rolls back earlier item');

ROLLBACK;
