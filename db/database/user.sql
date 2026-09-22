-- PostgreSQL user and role configuration for Agentrix
-- Creates the agentrix role if it does not already exist, and grants database creation privilege.

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'agentrix') THEN
        CREATE ROLE agentrix WITH LOGIN PASSWORD 'agentrix';
    ELSE
        ALTER ROLE agentrix WITH LOGIN PASSWORD 'agentrix';
    END IF;
END $$;

ALTER ROLE agentrix CREATEDB;
