BEGIN;
SET search_path = agentrix, pg_catalog;

SELECT test_support.assert_true(to_regnamespace('agentrix') IS NOT NULL,
    'canonical schema exists');
SELECT test_support.assert_true((
    SELECT count(*) >= 45 FROM information_schema.tables WHERE table_schema = 'agentrix'
), 'all domain tables were installed');
SELECT test_support.assert_true((
    SELECT count(*) >= 35 FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace
     WHERE n.nspname = 'agentrix' AND c.contype = 'f'
), 'cross-module foreign keys exist in the PostgreSQL catalog');
SELECT test_support.assert_true((
    SELECT count(*) >= 70 FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace
     WHERE n.nspname = 'agentrix' AND c.contype = 'c'
), 'range and state checks exist in the PostgreSQL catalog');

DO $test$
BEGIN
    BEGIN
        CREATE TABLE agentrix._atomic_probe(id integer PRIMARY KEY);
        RAISE EXCEPTION 'forced module failure';
    EXCEPTION WHEN OTHERS THEN
        NULL;
    END;
    PERFORM test_support.assert_true(to_regclass('agentrix._atomic_probe') IS NULL,
        'failed transactional module leaves no partial object');
END;
$test$;

ROLLBACK;

