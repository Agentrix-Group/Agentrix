-- PostgreSQL database initialization for Agentrix
-- Drops and recreates the agentrix database, assigning ownership to the agentrix role.

DROP DATABASE IF EXISTS agentrix;
CREATE DATABASE agentrix WITH OWNER agentrix;

GRANT ALL PRIVILEGES ON DATABASE agentrix TO agentrix;

\c agentrix
