BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT test_support.add_scored_fixture();
SELECT test_support.assert_true((SELECT count(*)=2 FROM judgements WHERE effective),
    'effective judgements consume the two exact submission seats');
SELECT test_support.expect_error($sql$
    INSERT INTO judgement_matches(judgement_id,submission_id,batch_id,match_id,seat_id,test_case_id)
    VALUES ('00000000-0000-0000-0000-000000000951','00000000-0000-0000-0000-000000000801',
      '00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000921',
      '00000000-0000-0000-0000-000000000932','00000000-0000-0000-0000-000000000751')
$sql$, '23514', 'does not contain');

INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,
 evaluation_policy_version_id,scoring_policy_version_id,score_scope,state,verdict,task_points,effective,judged_at)
SELECT '50000000-0000-0000-0000-000000000001',contest_id,contest_task_id,id,
 '00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000811',
 '00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000402',
 'official','awaiting_verification','accepted',70,false,now()
FROM submissions WHERE id='00000000-0000-0000-0000-000000000801';
SELECT test_support.expect_error($sql$
    SELECT verify_judgement('50000000-0000-0000-0000-000000000001',
      '00000000-0000-0000-0000-000000000004',now())
$sql$, '42501', 'not an active contest juror');
SELECT verify_judgement('50000000-0000-0000-0000-000000000001',
 '00000000-0000-0000-0000-000000000003',now());
SELECT test_support.assert_true((SELECT state='verified' FROM judgements
    WHERE id='50000000-0000-0000-0000-000000000001'), 'jury verification is explicit');
SELECT test_support.expect_error($sql$
    SELECT make_judgement_effective('50000000-0000-0000-0000-000000000001',now(),'double')
$sql$, '23514', 'at least one match');
SELECT test_support.expect_error($sql$
    UPDATE judgements SET effective=true,effective_at=now()
     WHERE id='50000000-0000-0000-0000-000000000001'
$sql$, '23505', NULL);

INSERT INTO submissions(id,contest_id,contest_entry_id,contest_task_id,submitted_by_user_id,
 source_artifact_id,toolchain_id,submitted_at,received_at,idempotency_key,disposition) VALUES
('50000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000732','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000211','00000000-0000-0000-0000-000000000502',now(),now(),'compile-fixture','eligible');
INSERT INTO build_attempts(id,submission_id,attempt_number,toolchain_id,state,failure_kind,started_at,ended_at,created_at) VALUES
('50000000-0000-0000-0000-000000000011','50000000-0000-0000-0000-000000000010',1,'00000000-0000-0000-0000-000000000502','failed','source',now(),now(),now());
INSERT INTO evaluation_batches(id,contest_id,contest_task_id,game_release_id,evaluation_policy_version_id,
 scoring_policy_version_id,purpose,reference_kind,state,roster_digest,created_by_user_id,created_at,sealed_at,completed_at) VALUES
('50000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000732','00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000403','official','fixed_suite','completed',decode(repeat('62',32),'hex'),'00000000-0000-0000-0000-000000000003',now(),now(),now());
INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,
 evaluation_policy_version_id,scoring_policy_version_id,score_scope,state,verdict,effective,judged_at) VALUES
('50000000-0000-0000-0000-000000000013','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000732','50000000-0000-0000-0000-000000000010','50000000-0000-0000-0000-000000000012','50000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000403','official','complete','compile_error',false,now());
SELECT make_judgement_effective('50000000-0000-0000-0000-000000000013',now(),'compile-effective');
SELECT test_support.assert_true((SELECT effective FROM judgements WHERE id='50000000-0000-0000-0000-000000000013'), 'failed compilation yields an effective judgement without a match');

ROLLBACK;
