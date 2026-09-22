BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT test_support.assert_true(NOT EXISTS (
    SELECT 1 FROM build_attempts WHERE submission_id='00000000-0000-0000-0000-000000000801' AND attempt_number=2
), 'submission may exist without a new build');
INSERT INTO build_attempts(id, submission_id, attempt_number, toolchain_id, state,
    failure_kind, started_at, ended_at, created_at)
VALUES ('20000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000801',2,
    '00000000-0000-0000-0000-000000000502','failed','source',now(),now(),now());
SELECT test_support.assert_true((SELECT state='failed' AND executable_artifact_id IS NULL
    FROM build_attempts WHERE id='20000000-0000-0000-0000-000000000001'),
    'failed build preserves technical failure without executable');
INSERT INTO build_attempts(id, submission_id, attempt_number, toolchain_id, state,
    executable_artifact_id, started_at, ended_at, created_at)
VALUES ('20000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000801',3,
    '00000000-0000-0000-0000-000000000502','succeeded','00000000-0000-0000-0000-000000000207',now(),now(),now());
SELECT test_support.expect_error($sql$
    UPDATE submissions SET source_artifact_id='00000000-0000-0000-0000-000000000206'
     WHERE id='00000000-0000-0000-0000-000000000801'
$sql$, '55000', 'immutable');

UPDATE artifacts SET publication_state='failed', failure_reason='upload failed'
 WHERE id='00000000-0000-0000-0000-000000000210';
SELECT test_support.expect_error($sql$
    UPDATE artifacts SET publication_state='ready', failure_reason=NULL,
      sha256=decode(repeat('61',32),'hex'), size_bytes=1, storage_key='late', verified_at=now()
     WHERE id='00000000-0000-0000-0000-000000000210'
$sql$, '55000', 'terminal state');
SELECT test_support.expect_error($sql$
    INSERT INTO artifacts(id,kind,publication_state,sha256,size_bytes,media_type,storage_key,created_at,verified_at)
    VALUES ('20000000-0000-0000-0000-000000000003','source','ready',decode('01','hex'),1,'text/plain','bad',now(),now())
$sql$, '23514', NULL);

ROLLBACK;

