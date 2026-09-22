BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT test_support.expect_error($sql$
 INSERT INTO batch_roster(id,batch_id,contest_id,contest_task_id,source_kind,contest_entry_id,
   submission_id,build_attempt_id,executable_artifact_id,executable_digest,added_at)
 VALUES ('30000000-0000-0000-0000-000000000000','00000000-0000-0000-0000-000000000901',
   '00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','submission',
   '00000000-0000-0000-0000-000000000722','00000000-0000-0000-0000-000000000801',
   '00000000-0000-0000-0000-000000000811','00000000-0000-0000-0000-000000000207',
   decode(repeat('07',32),'hex'),now())
$sql$, '23503', NULL);
INSERT INTO match_seats(id,match_id,batch_id,roster_item_id,seat_index,created_at)
VALUES ('30000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000921',
 '00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000911',2,now());
SELECT seal_evaluation_batch('00000000-0000-0000-0000-000000000901',decode(repeat('41',32),'hex'),now());
SELECT test_support.expect_error($sql$
    SELECT seal_match('00000000-0000-0000-0000-000000000921',now())
$sql$, '23514', 'duplicate program');
-- Failed seal rolled back its own transition; remove the plausible bad seat and seal.
DELETE FROM match_seats WHERE id='30000000-0000-0000-0000-000000000001';
SELECT seal_match('00000000-0000-0000-0000-000000000921',now());
SELECT test_support.expect_error($sql$
    UPDATE match_seats SET roster_item_id='00000000-0000-0000-0000-000000000913'
     WHERE id='00000000-0000-0000-0000-000000000931'
$sql$, '55000', 'immutable');
SELECT test_support.expect_error($sql$
    INSERT INTO batch_roster(id,batch_id,contest_id,contest_task_id,source_kind,
      baseline_program_id,executable_artifact_id,executable_digest,added_at)
    VALUES ('30000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000901',
      '00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','baseline',
      '00000000-0000-0000-0000-000000000601','00000000-0000-0000-0000-000000000204',
      decode(repeat('04',32),'hex'),now())
$sql$, '55000', 'immutable');
SELECT test_support.assert_true((SELECT seed=42 AND state='sealed' FROM matches
    WHERE id='00000000-0000-0000-0000-000000000921'),
    'seed and sealed specification remain frozen');

INSERT INTO matches(id,batch_id,contest_id,contest_task_id,game_release_id,test_case_id,purpose,seed,
 schedule_key,specification_digest,allow_duplicate_programs,state,created_at) VALUES
('30000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000751','official',44,'too-few-seats',decode(repeat('45',32),'hex'),false,'planned',now());
INSERT INTO match_seats(id,match_id,batch_id,roster_item_id,seat_index,created_at) VALUES
('30000000-0000-0000-0000-000000000011','30000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000901','00000000-0000-0000-0000-000000000913',0,now());
SELECT test_support.expect_error($sql$
 SELECT seal_match('30000000-0000-0000-0000-000000000010',now())
$sql$, '23514', 'outside [2,4]');
SELECT test_support.expect_error($sql$
 INSERT INTO matches(id,batch_id,contest_id,contest_task_id,game_release_id,purpose,seed,
   schedule_key,specification_digest,allow_duplicate_programs,state,created_at)
 VALUES ('30000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000901',
   '00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731',
   '30000000-0000-0000-0000-000000000099','official',45,'wrong-release',decode(repeat('46',32),'hex'),false,'planned',now())
$sql$, '23503', NULL);

ROLLBACK;
