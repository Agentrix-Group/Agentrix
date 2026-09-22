BEGIN;
SET search_path = agentrix, pg_catalog;

INSERT INTO sessions(id, user_id, family_id, token_digest, created_at, expires_at) VALUES
('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000010', decode(repeat('51',32),'hex'), now(), now()+interval '1 hour'),
('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000005', '10000000-0000-0000-0000-000000000011', decode(repeat('52',32),'hex'), now(), now()+interval '1 hour');
SELECT test_support.assert_true(authenticate_session(decode(repeat('51',32),'hex'), now()) = '00000000-0000-0000-0000-000000000001',
    'active session authenticates by digest');
SELECT test_support.assert_true(authenticate_session(decode(repeat('52',32),'hex'), now()) IS NULL,
    'disabled user session fails closed');
SELECT test_support.expect_error($sql$
    INSERT INTO users(id,username,email,status,created_at,updated_at) VALUES
    ('10000000-0000-0000-0000-000000000003','alice','new@example.test','active',now(),now())
$sql$, '23505', NULL);
INSERT INTO teams(id,slug,display_name,status,created_by_user_id,created_at) VALUES
('10000000-0000-0000-0000-000000000011','late-team','Late Team','active','00000000-0000-0000-0000-000000000004',now());
INSERT INTO team_memberships(id,team_id,user_id,member_role,joined_at,granted_by_user_id) VALUES
('10000000-0000-0000-0000-000000000012','10000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000004','owner',now()-interval '1 minute','00000000-0000-0000-0000-000000000004');
SELECT test_support.expect_error($sql$
 SELECT register_team_for_contest('10000000-0000-0000-0000-000000000013',
  '00000000-0000-0000-0000-000000000701','10000000-0000-0000-0000-000000000011',
  '00000000-0000-0000-0000-000000000711','00000000-0000-0000-0000-000000000004',NULL)
$sql$, '22023', 'late registration');
SELECT test_support.expect_error($sql$
 INSERT INTO contest_entries(id,contest_id,team_id,division_id,status,eligibility,registered_at)
 VALUES ('10000000-0000-0000-0000-000000000014','00000000-0000-0000-0000-000000000701',
  '10000000-0000-0000-0000-000000000011','00000000-0000-0000-0000-000000000713',
  'pending','pending_review',now())
$sql$, '23503', NULL);
SELECT test_support.expect_error($sql$
    INSERT INTO contest_entries(id,contest_id,team_id,division_id,status,eligibility,registered_at,accepted_at)
    VALUES ('10000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000701',
    '00000000-0000-0000-0000-000000000101','00000000-0000-0000-0000-000000000711','accepted','eligible',now(),now())
$sql$, '23505', NULL);

SELECT test_support.expect_error($sql$
    SELECT submit_program('10000000-0000-0000-0000-000000000005',
      '00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000731',
      '00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000211',
      '00000000-0000-0000-0000-000000000502','outsider')
$sql$, '42501', 'active member');
SELECT test_support.expect_error($sql$
    SELECT submit_program('10000000-0000-0000-0000-000000000006',
      '00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000733',
      '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000211',
      '00000000-0000-0000-0000-000000000502','cross-contest')
$sql$, '23514', 'different contests');
SELECT test_support.expect_error($sql$
    SELECT submit_program('10000000-0000-0000-0000-000000000007',
      '00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000731',
      '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000210',
      '00000000-0000-0000-0000-000000000502','pending-source')
$sql$, '22023', 'not ready');

SELECT submit_program('10000000-0000-0000-0000-000000000008',
  '00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000731',
  '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000211',
  '00000000-0000-0000-0000-000000000502','valid-once');
SELECT submit_program('10000000-0000-0000-0000-000000000009',
  '00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000731',
  '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000211',
  '00000000-0000-0000-0000-000000000502','valid-once');
SELECT test_support.assert_true((SELECT count(*) = 1 FROM submissions WHERE idempotency_key='valid-once'),
    'submission idempotency key creates one immutable version');

UPDATE contests SET submission_closes_at = now()-interval '1 minute', ends_at = now()+interval '1 day'
 WHERE id='00000000-0000-0000-0000-000000000701';
SELECT test_support.expect_error($sql$
    SELECT submit_program('10000000-0000-0000-0000-000000000010',
      '00000000-0000-0000-0000-000000000721','00000000-0000-0000-0000-000000000731',
      '00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000211',
      '00000000-0000-0000-0000-000000000502','late')
$sql$, '22023', 'window is closed');

ROLLBACK;
