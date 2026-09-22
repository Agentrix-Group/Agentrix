\set ON_ERROR_STOP on

DO $preflight$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'agentrix') THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'database is not empty: schema agentrix already exists';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname NOT LIKE 'pg_%'
           AND n.nspname <> 'information_schema'
           AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S')
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'database is not empty: user tables, views, sequences, or foreign tables exist';
    END IF;
END;
$preflight$;

