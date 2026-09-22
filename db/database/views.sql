-- PostgreSQL Views for Agentrix platform
-- Canonical filtered views for active and valid system records.

CREATE OR REPLACE VIEW roles_active AS
    SELECT * FROM roles WHERE active = TRUE;

CREATE OR REPLACE VIEW permissions_active AS
    SELECT * FROM permissions WHERE active = TRUE;

CREATE OR REPLACE VIEW role_permissions_active AS
    SELECT * FROM role_permissions WHERE active = TRUE;

CREATE OR REPLACE VIEW users_active AS
    SELECT * FROM users WHERE active = TRUE;

CREATE OR REPLACE VIEW categories_active AS
    SELECT * FROM categories WHERE active = TRUE;

CREATE OR REPLACE VIEW games_active AS
    SELECT * FROM games WHERE active = TRUE;

CREATE OR REPLACE VIEW contests_active AS
    SELECT * FROM contests WHERE active = TRUE;

CREATE OR REPLACE VIEW agents_active AS
    SELECT * FROM agents WHERE active = TRUE;

CREATE OR REPLACE VIEW submissions_active AS
    SELECT * FROM submissions WHERE active = TRUE;

CREATE OR REPLACE VIEW contest_entries_active AS
    SELECT * FROM contest_entries WHERE status != 'disqualified';

CREATE OR REPLACE VIEW matches_active AS
    SELECT * FROM matches WHERE active = TRUE;

CREATE OR REPLACE VIEW replays_active AS
    SELECT * FROM replays WHERE active = TRUE;
