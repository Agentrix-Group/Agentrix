SET search_path = agentrix, pg_catalog;

CREATE FUNCTION authenticate_session(p_token_digest sha256_digest, p_now timestamptz)
RETURNS uuid
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = agentrix, pg_catalog
AS $$
    SELECT s.user_id
      FROM sessions s JOIN users u ON u.id = s.user_id
     WHERE s.token_digest = p_token_digest
       AND s.revoked_at IS NULL AND s.expires_at > p_now
       AND u.status = 'active'
$$;

REVOKE ALL ON SCHEMA agentrix FROM PUBLIC;
REVOKE ALL ON ALL TABLES IN SCHEMA agentrix FROM PUBLIC;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA agentrix FROM PUBLIC;
REVOKE ALL ON ALL FUNCTIONS IN SCHEMA agentrix FROM PUBLIC;

