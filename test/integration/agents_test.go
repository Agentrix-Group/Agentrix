package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/stretchr/testify/require"
)

// Scenario 1: Fresh database without seeds.
// Expected: returns 404 (Game or User not found) with diagnostic and request_id.
func TestIntegration_Agents_FreshDatabaseWithoutSeeds_ClassifiedCorrectly(t *testing.T) {
	r := require.New(t)
	os.Setenv("MODE", "dev")
	defer os.Unsetenv("MODE")

	dbName := fmt.Sprintf("agentrix_test_agents_fresh_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// Initialize schema via migrations (NO SEEDS for games)
	r.NoError(database.Migrate(conn.Db))

	// Insert active user so request passes auth/permission check to reach database agent creation
	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id) 
		VALUES ('u-admin-01', 'admin', 'admin@agentrix.local', 'hash', 'admin');
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)
	tokens, err := authMod.GenerateAuthToken("u-admin-01", common.RoleAdmin)
	r.NoError(err)

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "AlphaBot",
		"game_id":     "starfighter",
		"description": "Bot created in unseeded database",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-agents-fresh-404")

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	// Must NOT be 500!
	r.NotEqual(http.StatusInternalServerError, rec.Code, "Integrity violation must not return 500")
	r.Equal(http.StatusNotFound, rec.Code, "Expected 404 Not Found for missing FK entity")

	var resp common.Response
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	r.NoError(err)
	r.Equal("req-agents-fresh-404", resp.RequestId, "Response must include request_id")
	r.NotEmpty(resp.Diagnostic, "Dev mode must include diagnostic details")
	r.Contains(resp.Diagnostic, "violated", "Diagnostic should explain constraint")
}

// Scenario 2: Legacy database where games table contains arena-basica instead of starfighter.
func TestIntegration_Agents_LegacyDatabase_ClassifiedCorrectly(t *testing.T) {
	r := require.New(t)
	os.Setenv("MODE", "dev")
	defer os.Unsetenv("MODE")

	dbName := fmt.Sprintf("agentrix_test_agents_legacy_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))

	// Insert legacy seeds
	_, err := conn.Db.Exec(`
		INSERT INTO roles (id, description) VALUES ('admin', 'Admin role') ON CONFLICT DO NOTHING;
		INSERT INTO users (id, username, email, password, role_id) 
		VALUES ('u-admin-01', 'admin', 'admin@agentrix.local', 'hash', 'admin');
		INSERT INTO games (id, name, description) VALUES ('arena-basica', 'Arena Basica', 'Legacy game');
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)
	tokens, err := authMod.GenerateAuthToken("u-admin-01", common.RoleAdmin)
	r.NoError(err)

	// Case A: starfighter submitted but missing in games table
	{
		body, _ := json.Marshal(map[string]interface{}{
			"name":    "StarBot",
			"game_id": "starfighter",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-Id", "req-legacy-starfighter-missing")

		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)

		r.NotEqual(http.StatusInternalServerError, rec.Code)
		r.Equal(http.StatusNotFound, rec.Code)

		var resp common.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		r.Equal("req-legacy-starfighter-missing", resp.RequestId)
		r.Contains(resp.Diagnostic, "agents_game_id_fkey")
	}

	// Case B: arena-basica submitted (unsupported game in MVP)
	{
		body, _ := json.Marshal(map[string]interface{}{
			"name":    "ArenaBot",
			"game_id": "arena-basica",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-Id", "req-legacy-unsupported")

		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)

		r.NotEqual(http.StatusInternalServerError, rec.Code)
		r.Equal(http.StatusBadRequest, rec.Code)

		var resp common.Response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		r.Equal("req-legacy-unsupported", resp.RequestId)
		r.Contains(resp.Message, "starfighter")
	}
}

// Scenario 3: Valid starfighter agent creation succeeds, duplicate returns 409 Conflict.
func TestIntegration_Agents_ValidAndDuplicate_ClassifiedCorrectly(t *testing.T) {
	r := require.New(t)
	os.Setenv("MODE", "dev")
	defer os.Unsetenv("MODE")

	dbName := fmt.Sprintf("agentrix_test_agents_valid_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))

	// Insert starfighter and user
	_, err := conn.Db.Exec(`
		INSERT INTO roles (id, description) VALUES ('admin', 'Admin role') ON CONFLICT DO NOTHING;
		INSERT INTO users (id, username, email, password, role_id) 
		VALUES ('u-admin-01', 'admin', 'admin@agentrix.local', 'hash', 'admin');
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Active game');
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)
	tokens, err := authMod.GenerateAuthToken("u-admin-01", common.RoleAdmin)
	r.NoError(err)

	// Create valid agent
	body, _ := json.Marshal(map[string]interface{}{
		"id":      "agent-deterministic-id-01",
		"name":    "StarBot",
		"game_id": "starfighter",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-create-valid")

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	r.Equal(http.StatusCreated, rec.Code)

	// Attempt duplicate insertion
	reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	reqDup.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
	reqDup.Header.Set("Content-Type", "application/json")
	reqDup.Header.Set("X-Request-Id", "req-create-duplicate")

	recDup := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recDup, reqDup)
	r.NotEqual(http.StatusInternalServerError, recDup.Code)
	r.Equal(http.StatusConflict, recDup.Code)

	var respDup common.Response
	_ = json.Unmarshal(recDup.Body.Bytes(), &respDup)
	r.Equal("req-create-duplicate", respDup.RequestId)
	r.Contains(respDup.Diagnostic, "duplicate entry")
}

// Scenario 4: In production mode, diagnostics are hidden from user while request_id is preserved.
func TestIntegration_Agents_ProductionModeMasksDiagnostics(t *testing.T) {
	r := require.New(t)
	os.Setenv("MODE", "production")
	defer os.Unsetenv("MODE")

	dbName := fmt.Sprintf("agentrix_test_agents_prod_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))

	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id) 
		VALUES ('u-admin-01', 'admin', 'admin@agentrix.local', 'hash', 'admin');
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)
	tokens, err := authMod.GenerateAuthToken("u-admin-01", common.RoleAdmin)
	r.NoError(err)

	body, _ := json.Marshal(map[string]interface{}{
		"name":    "AlphaBot",
		"game_id": "starfighter",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(body))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-prod-safe-id")

	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	r.Equal(http.StatusNotFound, rec.Code)
	var resp common.Response
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	r.Equal("req-prod-safe-id", resp.RequestId)
	r.Empty(resp.Diagnostic, "Production mode must NOT leak internal diagnostic info")
}
