BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT seal_evaluation_batch('00000000-0000-0000-0000-000000000901',decode(repeat('41',32),'hex'),now());
SELECT seal_match('00000000-0000-0000-0000-000000000921',now());
SELECT enqueue_match('00000000-0000-0000-0000-000000000941','00000000-0000-0000-0000-000000000921',10,now());
SELECT claim_match_job('00000000-0000-0000-0000-000000000942','worker-a');
SELECT record_match_seat_result('00000000-0000-0000-0000-000000000942',1,
 '00000000-0000-0000-0000-000000000931','completed',60,1,NULL);
SELECT record_match_seat_result('00000000-0000-0000-0000-000000000942',1,
 '00000000-0000-0000-0000-000000000932','completed',50,2,NULL);
SELECT record_match_replay('00000000-0000-0000-0000-000000000942',1,
 '00000000-0000-0000-0000-000000000209','agentrix-replay','1',1000);
SELECT test_support.expect_error($sql$
    SELECT accept_match_attempt('00000000-0000-0000-0000-000000000942',1,
      decode(repeat('02',32),'hex'),decode(repeat('03',32),'hex'),'1',decode(repeat('42',32),'hex'))
$sql$, '23514', 'verified replay');
SELECT verify_match_replay('00000000-0000-0000-0000-000000000942',now());
SELECT accept_match_attempt('00000000-0000-0000-0000-000000000942',1,
 decode(repeat('02',32),'hex'),decode(repeat('03',32),'hex'),'1',decode(repeat('42',32),'hex'));
SELECT test_support.assert_true((SELECT accepted_attempt_id='00000000-0000-0000-0000-000000000942'
    FROM matches WHERE id='00000000-0000-0000-0000-000000000921'), 'one attempt is accepted atomically');
SELECT test_support.expect_error($sql$
    SELECT record_match_seat_result('00000000-0000-0000-0000-000000000942',1,
      '00000000-0000-0000-0000-000000000931','completed',99,1,NULL)
$sql$, '40001', 'stale');
SELECT test_support.expect_error($sql$
    INSERT INTO match_seat_results(match_id,attempt_id,seat_id,outcome,engine_score,placement,created_at)
    VALUES ('30000000-0000-0000-0000-000000000099','00000000-0000-0000-0000-000000000942',
      '00000000-0000-0000-0000-000000000931','completed',1,1,now())
$sql$, '23503', NULL);

-- Expired lease creates a new fenced attempt; infrastructure failure does not
-- create a competitive seat result or defeat.
INSERT INTO matches(id,batch_id,contest_id,contest_task_id,game_release_id,test_case_id,purpose,seed,
 schedule_key,specification_digest,allow_duplicate_programs,state,created_at) VALUES
('40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000751','official',43,'retry-match',decode(repeat('43',32),'hex'),false,'planned',now());
INSERT INTO match_seats(id,match_id,batch_id,roster_item_id,seat_index,created_at) VALUES
('40000000-0000-0000-0000-000000000002','40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000911',0,now()),
('40000000-0000-0000-0000-000000000003','40000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000912',1,now());
SELECT seal_match('40000000-0000-0000-0000-000000000001',now());
SELECT enqueue_match('40000000-0000-0000-0000-000000000004','40000000-0000-0000-0000-000000000001',1,now());
SELECT claim_match_job('40000000-0000-0000-0000-000000000005','worker-old');
UPDATE match_jobs SET lease_expires_at=clock_timestamp()-interval '1 second'
 WHERE id='40000000-0000-0000-0000-000000000004';
SELECT claim_match_job('40000000-0000-0000-0000-000000000006','worker-new');
SELECT test_support.expect_error($sql$
 SELECT record_match_seat_result('40000000-0000-0000-0000-000000000005',1,
   '40000000-0000-0000-0000-000000000002','completed',1,1,NULL)
$sql$, '40001', 'stale');
SELECT fail_match_attempt('40000000-0000-0000-0000-000000000006',2,'worker_infrastructure','worker_lost');
SELECT test_support.assert_true((SELECT count(*)=2 FROM match_attempts WHERE match_id='40000000-0000-0000-0000-000000000001'), 'expired lease preserves both technical attempts');
SELECT test_support.assert_true(NOT EXISTS (SELECT 1 FROM match_seat_results WHERE match_id='40000000-0000-0000-0000-000000000001'), 'infrastructure failure is not recorded as player defeat');

ROLLBACK;
