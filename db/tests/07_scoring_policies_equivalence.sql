BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT test_support.assert_true(valid_scoring_policy('icpc_pass_fail','{"wrong_submission_penalty_minutes":20}'::jsonb), 'ICPC policy contract is implemented');
SELECT test_support.assert_true(valid_scoring_policy('best_score','{"higher_is_better":true}'::jsonb), 'best-score policy contract is implemented');
SELECT test_support.assert_true(valid_scoring_policy('aggregate_points','{"higher_is_better":true,"group_aggregation":"sum"}'::jsonb), 'aggregate policy contract is implemented');
SELECT test_support.assert_true(valid_scoring_policy('league_points','{"win_points":3,"draw_points":1,"loss_points":0}'::jsonb), 'league policy contract is implemented');
SELECT test_support.expect_error($sql$
 INSERT INTO scoring_policy_versions(id,name,kind,algorithm_version,parameters,parameters_digest,score_unit,disqualification_task_points,tie_method,created_at)
 VALUES ('60000000-0000-0000-0000-000000000001','bad','best_score',9,'{"higher_is_better":false}',decode(repeat('71',32),'hex'),'task_points',0,'rank',now())
$sql$, '23514', NULL);

SELECT test_support.add_scored_fixture();

-- Additional fixture tasks exercise ICPC, aggregate case points, and league points.
INSERT INTO contest_tasks(id,contest_id,code,title,position,game_release_id,evaluation_policy_version_id,
 scoring_policy_version_id,max_task_points,allow_submit,allow_judge,requires_verified_replay,visibility,published_at) VALUES
('60000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000701','C','Aggregate',3,'00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000404',100,true,true,false,'public',now()),
('60000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000701','D','League',4,'00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000405',100,true,true,false,'public',now());
INSERT INTO task_toolchains(contest_id,task_id,toolchain_id,enabled) VALUES
('00000000-0000-0000-0000-000000000701','60000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000502',true),
('00000000-0000-0000-0000-000000000701','60000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000502',true);
INSERT INTO test_groups(id,contest_id,task_id,key,weight,aggregation,visibility) VALUES
('60000000-0000-0000-0000-000000000012','00000000-0000-0000-0000-000000000701','60000000-0000-0000-0000-000000000010','aggregate',1,'sum','private');
INSERT INTO test_cases(id,task_id,group_id,key,seed,weight,visibility) VALUES
('60000000-0000-0000-0000-000000000013','60000000-0000-0000-0000-000000000010','60000000-0000-0000-0000-000000000012','one',1,1,'private'),
('60000000-0000-0000-0000-000000000014','60000000-0000-0000-0000-000000000010','60000000-0000-0000-0000-000000000012','two',2,2,'private');

INSERT INTO evaluation_batches(id,contest_id,contest_task_id,game_release_id,evaluation_policy_version_id,
 scoring_policy_version_id,purpose,reference_kind,state,roster_digest,created_by_user_id,created_at,sealed_at,completed_at) VALUES
('60000000-0000-0000-0000-000000000020','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000732','00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000403','official','fixed_suite','completed',decode(repeat('72',32),'hex'),'00000000-0000-0000-0000-000000000003',now(),now(),now()),
('60000000-0000-0000-0000-000000000021','00000000-0000-0000-0000-000000000701','60000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000404','official','fixed_suite','completed',decode(repeat('73',32),'hex'),'00000000-0000-0000-0000-000000000003',now(),now(),now()),
('60000000-0000-0000-0000-000000000022','00000000-0000-0000-0000-000000000701','60000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000405','official','sealed_roster','completed',decode(repeat('74',32),'hex'),'00000000-0000-0000-0000-000000000003',now(),now(),now());

INSERT INTO submissions(id,contest_id,contest_entry_id,contest_task_id,submitted_by_user_id,source_artifact_id,
 toolchain_id,submitted_at,received_at,idempotency_key,disposition) VALUES
('60000000-0000-0000-0000-000000000030','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000732','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000205','00000000-0000-0000-0000-000000000502',now()-interval '60 minutes',now()-interval '60 minutes','icpc-wrong','eligible'),
('60000000-0000-0000-0000-000000000031','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000732','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000211','00000000-0000-0000-0000-000000000502',now()-interval '50 minutes',now()-interval '50 minutes','icpc-ok','eligible'),
('60000000-0000-0000-0000-000000000032','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000721','60000000-0000-0000-0000-000000000010','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000205','00000000-0000-0000-0000-000000000502',now()-interval '40 minutes',now()-interval '40 minutes','aggregate','eligible'),
('60000000-0000-0000-0000-000000000033','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000721','60000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000205','00000000-0000-0000-0000-000000000502',now()-interval '30 minutes',now()-interval '30 minutes','league','eligible');
INSERT INTO build_attempts(id,submission_id,attempt_number,toolchain_id,state,executable_artifact_id,started_at,ended_at,created_at)
SELECT ('60000000-0000-0000-0000-' || right(replace(id::text,'-',''),12))::uuid,id,1,
 '00000000-0000-0000-0000-000000000502','succeeded','00000000-0000-0000-0000-000000000207',now(),now(),now()
FROM submissions WHERE id::text LIKE '60000000-0000-0000-0000-00000000003%';

INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,
 evaluation_policy_version_id,scoring_policy_version_id,score_scope,state,verdict,task_points,
 penalty_seconds,wins,draws,losses,effective,judged_at,effective_at) VALUES
('60000000-0000-0000-0000-000000000040','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000732','60000000-0000-0000-0000-000000000030','60000000-0000-0000-0000-000000000020','60000000-0000-0000-0000-000000000030','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000403','official','complete','wrong_answer',0,0,0,0,0,true,now(),now()),
('60000000-0000-0000-0000-000000000041','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000732','60000000-0000-0000-0000-000000000031','60000000-0000-0000-0000-000000000020','60000000-0000-0000-0000-000000000031','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000403','official','complete','accepted',1,0,0,0,0,true,now(),now()),
('60000000-0000-0000-0000-000000000042','00000000-0000-0000-0000-000000000701','60000000-0000-0000-0000-000000000010','60000000-0000-0000-0000-000000000032','60000000-0000-0000-0000-000000000021','60000000-0000-0000-0000-000000000032','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000404','official','complete','accepted',99,0,0,0,0,true,now(),now()),
('60000000-0000-0000-0000-000000000043','00000000-0000-0000-0000-000000000701','60000000-0000-0000-0000-000000000011','60000000-0000-0000-0000-000000000033','60000000-0000-0000-0000-000000000022','60000000-0000-0000-0000-000000000033','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000405','official','complete','accepted',NULL,0,2,1,0,true,now(),now());
INSERT INTO judgement_cases(judgement_id,contest_task_id,test_case_id,verdict,task_points) VALUES
('60000000-0000-0000-0000-000000000042','60000000-0000-0000-0000-000000000010','60000000-0000-0000-0000-000000000013','accepted',30),
('60000000-0000-0000-0000-000000000042','60000000-0000-0000-0000-000000000010','60000000-0000-0000-0000-000000000014','accepted',20);

INSERT INTO submissions(id,contest_id,contest_entry_id,contest_task_id,submitted_by_user_id,source_artifact_id,
 toolchain_id,submitted_at,received_at,idempotency_key,disposition) VALUES
('60000000-0000-0000-0000-000000000044','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000731','00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000211','00000000-0000-0000-0000-000000000502',now()-interval '5 minutes',now()-interval '5 minutes','dq-fixture','eligible');
INSERT INTO build_attempts(id,submission_id,attempt_number,toolchain_id,state,executable_artifact_id,started_at,ended_at,created_at) VALUES
('60000000-0000-0000-0000-000000000045','60000000-0000-0000-0000-000000000044',1,'00000000-0000-0000-0000-000000000502','succeeded','00000000-0000-0000-0000-000000000207',now(),now(),now());
INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,evaluation_policy_version_id,
 scoring_policy_version_id,score_scope,state,verdict,task_points,effective,judged_at,effective_at) VALUES
('60000000-0000-0000-0000-000000000046','00000000-0000-0000-0000-000000000701','00000000-0000-0000-0000-000000000731','60000000-0000-0000-0000-000000000044','00000000-0000-0000-0000-000000000901','60000000-0000-0000-0000-000000000045','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000402','official','complete','disqualified',99,true,now(),now());

SELECT rebuild_scoreboard('60000000-0000-0000-0000-000000000050','00000000-0000-0000-0000-000000000701',NULL,now());
SELECT test_support.assert_true((SELECT task_points=1 AND solved AND penalty_seconds>0 FROM score_cells
 WHERE revision_id='60000000-0000-0000-0000-000000000050' AND contest_task_id='00000000-0000-0000-0000-000000000732'), 'ICPC selects first accepted and applies time plus wrong-attempt penalty');
SELECT test_support.assert_true((SELECT task_points=70 FROM score_cells
 WHERE revision_id='60000000-0000-0000-0000-000000000050' AND contest_task_id='60000000-0000-0000-0000-000000000010'), 'aggregate points apply case and group weights');
SELECT test_support.assert_true((SELECT task_points=7 FROM score_cells
 WHERE revision_id='60000000-0000-0000-0000-000000000050' AND contest_task_id='60000000-0000-0000-0000-000000000011'), 'league points derive from wins and draws, not engine score');
SELECT test_support.assert_true((SELECT selected_judgement_id='00000000-0000-0000-0000-000000000951' FROM score_cells
 WHERE revision_id='60000000-0000-0000-0000-000000000050' AND contest_task_id='00000000-0000-0000-0000-000000000731' AND contest_entry_id='00000000-0000-0000-0000-000000000721'), 'DQ is zero and cannot displace a valid best score');

SELECT change_submission_disposition('60000000-0000-0000-0000-000000000060','00000000-0000-0000-0000-000000000802','ignored','fixture incremental change','00000000-0000-0000-0000-000000000003');
SELECT rebuild_scoreboard_incremental('60000000-0000-0000-0000-000000000051','00000000-0000-0000-0000-000000000701',now());
SELECT rebuild_scoreboard('60000000-0000-0000-0000-000000000052','00000000-0000-0000-0000-000000000701',NULL,now());
SELECT test_support.assert_true(NOT EXISTS (
 (SELECT contest_entry_id,contest_task_id,selected_submission_id,selected_judgement_id,task_points,solved,penalty_seconds,wins,draws,losses FROM score_cells WHERE revision_id='60000000-0000-0000-0000-000000000051'
  EXCEPT
  SELECT contest_entry_id,contest_task_id,selected_submission_id,selected_judgement_id,task_points,solved,penalty_seconds,wins,draws,losses FROM score_cells WHERE revision_id='60000000-0000-0000-0000-000000000052')
 UNION ALL
 (SELECT contest_entry_id,contest_task_id,selected_submission_id,selected_judgement_id,task_points,solved,penalty_seconds,wins,draws,losses FROM score_cells WHERE revision_id='60000000-0000-0000-0000-000000000052'
  EXCEPT
  SELECT contest_entry_id,contest_task_id,selected_submission_id,selected_judgement_id,task_points,solved,penalty_seconds,wins,draws,losses FROM score_cells WHERE revision_id='60000000-0000-0000-0000-000000000051')
), 'incremental and full score-cell rebuilds are equivalent');

-- The same projection function demonstrates the configured RANK versus
-- DENSE_RANK gap after a tie (100, 90, 90, 80).
INSERT INTO teams(id,slug,display_name,status,created_by_user_id,created_at) VALUES
('60000000-0000-0000-0000-000000000101','rank-one','Rank One','active','00000000-0000-0000-0000-000000000001',now()),
('60000000-0000-0000-0000-000000000102','rank-two','Rank Two','active','00000000-0000-0000-0000-000000000001',now()),
('60000000-0000-0000-0000-000000000103','rank-three','Rank Three','active','00000000-0000-0000-0000-000000000001',now()),
('60000000-0000-0000-0000-000000000104','rank-four','Rank Four','active','00000000-0000-0000-0000-000000000001',now());
INSERT INTO contests(id,slug,title,state,registration_opens_at,registration_closes_at,submission_opens_at,
 starts_at,submission_closes_at,ends_at,late_registration_policy,entry_uniqueness_policy,
 code_visibility,private_feedback_policy,rank_method,created_by_user_id,created_at) VALUES
('60000000-0000-0000-0000-000000000110','rank-fixture','Rank Fixture','running',now()-interval '3 days',now()-interval '2 days',now()-interval '1 day',now()-interval '12 hours',now()+interval '1 day',now()+interval '2 days','reject','one_per_team','jury_only','hidden','rank','00000000-0000-0000-0000-000000000003',now());
INSERT INTO contest_divisions(id,contest_id,key,title,sort_order,award_eligible) VALUES
('60000000-0000-0000-0000-000000000111','60000000-0000-0000-0000-000000000110','open','Open',1,true);
INSERT INTO contest_entries(id,contest_id,team_id,division_id,status,eligibility,registered_at,accepted_at)
SELECT ('60000000-0000-0000-0000-00000000012'||n)::uuid,'60000000-0000-0000-0000-000000000110',
       ('60000000-0000-0000-0000-00000000010'||n)::uuid,'60000000-0000-0000-0000-000000000111',
       'accepted','eligible',now(),now() FROM generate_series(1,4) n;
INSERT INTO contest_tasks(id,contest_id,code,title,position,game_release_id,evaluation_policy_version_id,
 scoring_policy_version_id,max_task_points,allow_submit,allow_judge,requires_verified_replay,visibility,published_at) VALUES
('60000000-0000-0000-0000-000000000130','60000000-0000-0000-0000-000000000110','A','Rank task',1,'00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000402',100,true,true,false,'public',now());
INSERT INTO task_toolchains(contest_id,task_id,toolchain_id,enabled) VALUES
('60000000-0000-0000-0000-000000000110','60000000-0000-0000-0000-000000000130','00000000-0000-0000-0000-000000000502',true);
INSERT INTO evaluation_batches(id,contest_id,contest_task_id,game_release_id,evaluation_policy_version_id,
 scoring_policy_version_id,purpose,reference_kind,state,roster_digest,created_by_user_id,created_at,sealed_at,completed_at) VALUES
('60000000-0000-0000-0000-000000000131','60000000-0000-0000-0000-000000000110','60000000-0000-0000-0000-000000000130','00000000-0000-0000-0000-000000000302','00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000402','official','fixed_suite','completed',decode(repeat('75',32),'hex'),'00000000-0000-0000-0000-000000000003',now(),now(),now());
INSERT INTO submissions(id,contest_id,contest_entry_id,contest_task_id,submitted_by_user_id,source_artifact_id,
 toolchain_id,submitted_at,received_at,idempotency_key,disposition)
SELECT ('60000000-0000-0000-0000-00000000014'||n)::uuid,'60000000-0000-0000-0000-000000000110',
       ('60000000-0000-0000-0000-00000000012'||n)::uuid,'60000000-0000-0000-0000-000000000130',
       '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000205',
       '00000000-0000-0000-0000-000000000502',now(),now(),'rank-'||n,'eligible'
  FROM generate_series(1,4) n;
INSERT INTO build_attempts(id,submission_id,attempt_number,toolchain_id,state,executable_artifact_id,started_at,ended_at,created_at)
SELECT ('60000000-0000-0000-0000-00000000015'||n)::uuid,('60000000-0000-0000-0000-00000000014'||n)::uuid,1,
       '00000000-0000-0000-0000-000000000502','succeeded','00000000-0000-0000-0000-000000000207',now(),now(),now()
  FROM generate_series(1,4) n;
INSERT INTO judgements(id,contest_id,contest_task_id,submission_id,batch_id,build_attempt_id,
 evaluation_policy_version_id,scoring_policy_version_id,score_scope,state,verdict,task_points,effective,judged_at,effective_at)
SELECT ('60000000-0000-0000-0000-00000000016'||n)::uuid,'60000000-0000-0000-0000-000000000110','60000000-0000-0000-0000-000000000130',
       ('60000000-0000-0000-0000-00000000014'||n)::uuid,'60000000-0000-0000-0000-000000000131',
       ('60000000-0000-0000-0000-00000000015'||n)::uuid,'00000000-0000-0000-0000-000000000401','00000000-0000-0000-0000-000000000402',
       'official','complete','accepted',(ARRAY[100,90,90,80])[n],true,now(),now()
  FROM generate_series(1,4) n;
SELECT rebuild_scoreboard('60000000-0000-0000-0000-000000000170','60000000-0000-0000-0000-000000000110',NULL,now());
SELECT test_support.assert_true((SELECT rank=4 FROM score_rows WHERE revision_id='60000000-0000-0000-0000-000000000170' AND contest_entry_id='60000000-0000-0000-0000-000000000124'), 'RANK leaves a gap after tied second place');
UPDATE contests SET rank_method='dense_rank' WHERE id='60000000-0000-0000-0000-000000000110';
SELECT rebuild_scoreboard('60000000-0000-0000-0000-000000000171','60000000-0000-0000-0000-000000000110',NULL,now());
SELECT test_support.assert_true((SELECT rank=3 FROM score_rows WHERE revision_id='60000000-0000-0000-0000-000000000171' AND contest_entry_id='60000000-0000-0000-0000-000000000124'), 'DENSE_RANK removes the gap while keeping ties');

ROLLBACK;
