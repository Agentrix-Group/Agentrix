-- PostgreSQL DDL for Agentrix: Contests table
-- Represents the Contest entity aligned with the Blueprint domain model.

CREATE TABLE IF NOT EXISTS contests (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    game_id VARCHAR(64),
    category_id VARCHAR(64),
    state VARCHAR(32) NOT NULL DEFAULT 'draft',
    status VARCHAR(32) DEFAULT 'upcoming',
    active BOOLEAN DEFAULT TRUE,
    starts_at TIMESTAMPTZ NULL,
    ends_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_contests_state ON contests(state);
CREATE INDEX IF NOT EXISTS idx_contests_created_at ON contests(created_at DESC);
