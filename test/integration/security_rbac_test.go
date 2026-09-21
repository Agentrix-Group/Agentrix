package integration

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/server"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

func init() {
	// Use fast Argon2 parameters in tests to keep test runs blazing fast
	auth.SetActiveArgon2Params(auth.FastArgon2Params)
}

// 1. RBAC Permissions Matrix: Multi-role resolution, dynamic deactivation, and inactive accounts
func TestIntegration_Security_RBAC_MultiRoleAndRevocation(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_sec_rbac_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(database.TargetSchemaVersion, ver)

	_, err = conn.Db.Exec(`
		INSERT INTO games (id, name, description, active) VALUES ('starfighter', 'Starfighter Arena', 'Active game', TRUE) ON CONFLICT DO NOTHING;
	`)
	r.NoError(err)

	srv, _ := setupTestServer(conn)

	// Hash password using Argon2id
	argonHash, err := auth.HashPassword("TestPass123!")
	r.NoError(err)

	// Create user with primary role 'spectator', but also assign 'player' role in user_roles
	_, err = conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-multirole-01', 'alice_multi', 'alice@agentrix.local', $1, 'spectator', TRUE)
	`, argonHash)
	r.NoError(err)
	_, err = conn.Db.Exec(`
		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-multirole-01', 'spectator'),
		('u-multirole-01', 'player')
	`)
	r.NoError(err)

	// Login as Alice
	loginBody, _ := json.Marshal(map[string]string{
		"username": "alice_multi",
		"password": "TestPass123!",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogin, reqLogin)
	r.Equal(http.StatusOK, recLogin.Code)

	var loginResp server.LoginResponse
	r.NoError(json.Unmarshal(recLogin.Body.Bytes(), &loginResp))
	r.NotNil(loginResp.Token)
	r.NotEmpty(loginResp.Token.AccessToken)
	aliceToken := loginResp.Token.AccessToken

	// Alice can create an agent because she has the active 'player' role
	agentBody, _ := json.Marshal(map[string]interface{}{
		"id":      "agent-alice-01",
		"name":    "AliceViper",
		"game_id": "starfighter",
	})
	reqAgent := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(agentBody))
	reqAgent.Header.Set("Authorization", fmt.Sprintf("Bearer %s", aliceToken))
	reqAgent.Header.Set("Content-Type", "application/json")
	recAgent := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recAgent, reqAgent)
	r.Equal(http.StatusCreated, recAgent.Code, "Alice must be allowed to create agent with active player role")

	// Dynamic role revocation: Revoke Alice's 'player' role in DB
	_, err = conn.Db.Exec(`
		DELETE FROM user_roles WHERE user_id = 'u-multirole-01' AND role_id = 'player';
	`)
	r.NoError(err)
	service.InvalidatePermissionCache()

	// Alice attempts to create another agent: Must immediately be rejected with 403 Forbidden!
	agentBody2, _ := json.Marshal(map[string]interface{}{
		"id":      "agent-alice-02",
		"name":    "AliceViper2",
		"game_id": "starfighter",
	})
	reqAgent2 := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(agentBody2))
	reqAgent2.Header.Set("Authorization", fmt.Sprintf("Bearer %s", aliceToken))
	reqAgent2.Header.Set("Content-Type", "application/json")
	recAgent2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recAgent2, reqAgent2)
	r.Equal(http.StatusForbidden, recAgent2.Code, "Deactivated role must immediately deny permission")

	// Account deactivation: Disable Alice's account entirely
	_, err = conn.Db.Exec(`
		UPDATE users SET active = FALSE WHERE id = 'u-multirole-01';
	`)
	r.NoError(err)
	service.InvalidatePermissionCache()

	// Alice tries to access /me with existing token -> rejected with 403
	reqMe := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	reqMe.Header.Set("Authorization", fmt.Sprintf("Bearer %s", aliceToken))
	recMe := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recMe, reqMe)
	r.Equal(http.StatusForbidden, recMe.Code, "Inactive user account must be denied access")

	// Alice tries to login -> rejected with 403
	reqLogin2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin2.Header.Set("Content-Type", "application/json")
	recLogin2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogin2, reqLogin2)
	r.Equal(http.StatusForbidden, recLogin2.Code, "Inactive user account cannot authenticate")

	_ = ctx
}

// 2. Scoped Data Access: Horizontal privilege escalation prevention
func TestIntegration_Security_ScopedDataAccess(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_sec_scope_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	srv, _ := setupTestServer(conn)

	argonHash, _ := auth.HashPassword("TestPass123!")

	// Create Charlie (player), Bob (player), and Eve (admin)
	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-charlie', 'charlie', 'charlie@agentrix.local', $1, 'player', TRUE),
		('u-bob', 'bob', 'bob@agentrix.local', $1, 'player', TRUE),
		('u-eve', 'eve', 'eve@agentrix.local', $1, 'admin', TRUE)
	`, argonHash)
	r.NoError(err)
	_, err = conn.Db.Exec(`
		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-charlie', 'player'),
		('u-bob', 'player'),
		('u-eve', 'admin')
	`)
	r.NoError(err)

	// Login Charlie
	loginCharlie, _ := json.Marshal(map[string]string{"username": "charlie", "password": "TestPass123!"})
	recC := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recC, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginCharlie)))
	r.Equal(http.StatusOK, recC.Code)
	var respC server.LoginResponse
	r.NoError(json.Unmarshal(recC.Body.Bytes(), &respC))
	r.NotNil(respC.Token)
	charlieToken := respC.Token.AccessToken

	// Login Bob
	loginBob, _ := json.Marshal(map[string]string{"username": "bob", "password": "TestPass123!"})
	recB := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recB, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBob)))
	r.Equal(http.StatusOK, recB.Code)
	var respB server.LoginResponse
	r.NoError(json.Unmarshal(recB.Body.Bytes(), &respB))
	r.NotNil(respB.Token)
	bobToken := respB.Token.AccessToken

	// Login Eve (Admin)
	loginEve, _ := json.Marshal(map[string]string{"username": "eve", "password": "TestPass123!"})
	recE := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recE, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginEve)))
	r.Equal(http.StatusOK, recE.Code)
	var respE server.LoginResponse
	r.NoError(json.Unmarshal(recE.Body.Bytes(), &respE))
	r.NotNil(respE.Token)
	eveToken := respE.Token.AccessToken

	// 1. Charlie reads own profile -> 200 OK
	reqC := httptest.NewRequest(http.MethodGet, "/api/v1/users/u-charlie", nil)
	reqC.Header.Set("Authorization", fmt.Sprintf("Bearer %s", charlieToken))
	recC2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recC2, reqC)
	r.Equal(http.StatusOK, recC2.Code, "Charlie can read own profile")

	// 2. Bob attempts to read Charlie's profile -> 403 Forbidden (Bob only has users:read:own)
	reqBobReadCharlie := httptest.NewRequest(http.MethodGet, "/api/v1/users/u-charlie", nil)
	reqBobReadCharlie.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bobToken))
	recBobReadCharlie := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recBobReadCharlie, reqBobReadCharlie)
	r.Equal(http.StatusForbidden, recBobReadCharlie.Code, "Bob cannot read Charlie's profile without users:read:any")

	// 3. Bob attempts to list all users -> 403 Forbidden
	reqBobList := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	reqBobList.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bobToken))
	recBobList := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recBobList, reqBobList)
	r.Equal(http.StatusForbidden, recBobList.Code, "Player cannot list user directory")

	// 4. Eve (admin) reads Charlie's profile -> 200 OK
	reqEveReadCharlie := httptest.NewRequest(http.MethodGet, "/api/v1/users/u-charlie", nil)
	reqEveReadCharlie.Header.Set("Authorization", fmt.Sprintf("Bearer %s", eveToken))
	recEveReadCharlie := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recEveReadCharlie, reqEveReadCharlie)
	r.Equal(http.StatusOK, recEveReadCharlie.Code, "Admin can read any user profile")

	// 5. Eve lists all users -> 200 OK
	reqEveList := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	reqEveList.Header.Set("Authorization", fmt.Sprintf("Bearer %s", eveToken))
	recEveList := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recEveList, reqEveList)
	r.Equal(http.StatusOK, recEveList.Code, "Admin can list all users")
}

// 3. Stateful Session Lifecycle: login -> refresh -> rotate -> revoke
func TestIntegration_Security_StatefulSessionLifecycle(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_sec_session_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	srv, _ := setupTestServer(conn)

	argonHash, _ := auth.HashPassword("SessionPass123!")
	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-session-01', 'dan_session', 'dan@agentrix.local', $1, 'player', TRUE)
	`, argonHash)
	r.NoError(err)
	_, err = conn.Db.Exec(`
		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-session-01', 'player')
	`)
	r.NoError(err)

	// Step 1: Login
	loginBody, _ := json.Marshal(map[string]string{
		"username": "dan_session",
		"password": "SessionPass123!",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin.Header.Set("User-Agent", "IntegrationTest/1.0")
	recLogin := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogin, reqLogin)
	r.Equal(http.StatusOK, recLogin.Code)

	var loginResp server.LoginResponse
	r.NoError(json.Unmarshal(recLogin.Body.Bytes(), &loginResp))
	r.NotNil(loginResp.Token)
	rt1 := loginResp.Token.RefreshToken
	r.NotEmpty(rt1)
	r.Contains(rt1, "artx_rf_")

	// Verify session row in database
	var countActive int
	err = conn.Db.QueryRow(`
		SELECT COUNT(*) FROM sessions WHERE user_id = 'u-session-01' AND is_revoked = FALSE
	`).Scan(&countActive)
	r.NoError(err)
	r.Equal(1, countActive, "Exactly one active session in DB after login")

	// Step 2: Refresh token (Single-use rotation)
	refreshBody, _ := json.Marshal(map[string]string{
		"refresh_token": rt1,
	})
	reqRefresh := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	recRefresh := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recRefresh, reqRefresh)
	r.Equal(http.StatusOK, recRefresh.Code)

	var refreshResp model.Token
	r.NoError(json.Unmarshal(recRefresh.Body.Bytes(), &refreshResp))
	rt2 := refreshResp.RefreshToken
	r.NotEmpty(rt2)
	r.NotEqual(rt1, rt2, "Refreshed token must be a new cryptographically rotated token")

	// In DB: rt1 must now be revoked, rt2 active
	var totalSessions, revokedSessions int
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = 'u-session-01'`).Scan(&totalSessions)
	r.NoError(err)
	r.Equal(2, totalSessions)
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = 'u-session-01' AND is_revoked = TRUE`).Scan(&revokedSessions)
	r.NoError(err)
	r.Equal(1, revokedSessions, "First session must be marked revoked after rotation")

	// Step 3: Logout
	logoutBody, _ := json.Marshal(map[string]string{
		"refresh_token": rt2,
	})
	reqLogout := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(logoutBody))
	recLogout := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogout, reqLogout)
	r.Equal(http.StatusOK, recLogout.Code)

	// In DB: active sessions should now be 0
	err = conn.Db.QueryRow(`
		SELECT COUNT(*) FROM sessions WHERE user_id = 'u-session-01' AND is_revoked = FALSE
	`).Scan(&countActive)
	r.NoError(err)
	r.Equal(0, countActive, "No active sessions remaining after logout")

	// Step 4: Refreshing with revoked rt2 must fail
	reqRefresh2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(logoutBody))
	recRefresh2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recRefresh2, reqRefresh2)
	r.Equal(http.StatusUnauthorized, recRefresh2.Code, "Revoked session cannot be refreshed")
}

// 4. Refresh Token Reuse Detection: Invalidates entire family immediately
func TestIntegration_Security_RefreshTokenReuseDetection(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_sec_reuse_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	srv, _ := setupTestServer(conn)

	argonHash, _ := auth.HashPassword("ReusePass123!")
	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-reuse-01', 'reuse_pilot', 'reuse@agentrix.local', $1, 'player', TRUE)
	`, argonHash)
	r.NoError(err)
	_, err = conn.Db.Exec(`
		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-reuse-01', 'player')
	`)
	r.NoError(err)

	// Step 1: Login -> rt1
	loginBody, _ := json.Marshal(map[string]string{
		"username": "reuse_pilot",
		"password": "ReusePass123!",
	})
	recLogin := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogin, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody)))
	r.Equal(http.StatusOK, recLogin.Code)

	var loginResp server.LoginResponse
	r.NoError(json.Unmarshal(recLogin.Body.Bytes(), &loginResp))
	r.NotNil(loginResp.Token)
	rt1 := loginResp.Token.RefreshToken

	// Step 2: First legitimate rotation: rt1 -> rt2
	refreshBody1, _ := json.Marshal(map[string]string{"refresh_token": rt1})
	recRef1 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recRef1, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody1)))
	r.Equal(http.StatusOK, recRef1.Code)

	var refResp1 model.Token
	r.NoError(json.Unmarshal(recRef1.Body.Bytes(), &refResp1))
	rt2 := refResp1.RefreshToken

	// Step 3: Attacker tries to replay already-rotated rt1!
	recReplay := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recReplay, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody1)))
	r.Equal(http.StatusUnauthorized, recReplay.Code, "Replay of rotated token must fail")

	// Step 4: Verify REUSE DETECTION revoked the entire family:
	// The legitimate user attempting to use rt2 must now also be rejected!
	refreshBody2, _ := json.Marshal(map[string]string{"refresh_token": rt2})
	recRef2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recRef2, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody2)))
	r.Equal(http.StatusUnauthorized, recRef2.Code, "Legitimate session in compromised family must be revoked")

	var activeInFamily int
	err = conn.Db.QueryRow(`
		SELECT COUNT(*) FROM sessions WHERE user_id = 'u-reuse-01' AND is_revoked = FALSE
	`).Scan(&activeInFamily)
	r.NoError(err)
	r.Equal(0, activeInFamily, "Entire token family must be revoked upon token reuse detection")
}

// 5. Global Logout: Revoke all user sessions
func TestIntegration_Security_GlobalLogout(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_sec_logoutall_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	srv, _ := setupTestServer(conn)

	argonHash, _ := auth.HashPassword("GlobalPass123!")
	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-global-01', 'global_pilot', 'global@agentrix.local', $1, 'player', TRUE)
	`, argonHash)
	r.NoError(err)
	_, err = conn.Db.Exec(`
		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-global-01', 'player')
	`)
	r.NoError(err)

	loginBody, _ := json.Marshal(map[string]string{
		"username": "global_pilot",
		"password": "GlobalPass123!",
	})

	// Login Session 1 (e.g. Chrome)
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req1.Header.Set("User-Agent", "Chrome/120.0")
	rec1 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec1, req1)
	r.Equal(http.StatusOK, rec1.Code)
	var resp1 server.LoginResponse
	r.NoError(json.Unmarshal(rec1.Body.Bytes(), &resp1))
	r.NotNil(resp1.Token)
	rt1 := resp1.Token.RefreshToken
	at1 := resp1.Token.AccessToken

	// Login Session 2 (e.g. Firefox)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	req2.Header.Set("User-Agent", "Firefox/122.0")
	rec2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec2, req2)
	r.Equal(http.StatusOK, rec2.Code)
	var resp2 server.LoginResponse
	r.NoError(json.Unmarshal(rec2.Body.Bytes(), &resp2))
	r.NotNil(resp2.Token)
	rt2 := resp2.Token.RefreshToken

	var activeBefore int
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = 'u-global-01' AND is_revoked = FALSE`).Scan(&activeBefore)
	r.NoError(err)
	r.Equal(2, activeBefore)

	// Call logout-all with at1
	reqLogoutAll := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout-all", nil)
	reqLogoutAll.Header.Set("Authorization", fmt.Sprintf("Bearer %s", at1))
	recLogoutAll := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogoutAll, reqLogoutAll)
	r.Equal(http.StatusOK, recLogoutAll.Code)

	var activeAfter int
	err = conn.Db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = 'u-global-01' AND is_revoked = FALSE`).Scan(&activeAfter)
	r.NoError(err)
	r.Equal(0, activeAfter, "All sessions must be revoked after logout-all")

	// Neither rt1 nor rt2 can refresh
	recRef1 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recRef1, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(fmt.Sprintf(`{"refresh_token":"%s"}`, rt1)))))
	r.Equal(http.StatusUnauthorized, recRef1.Code)

	recRef2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recRef2, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader([]byte(fmt.Sprintf(`{"refresh_token":"%s"}`, rt2)))))
	r.Equal(http.StatusUnauthorized, recRef2.Code)
}

// 6. Concurrent Session Pruning: Enforces maximum active session limit per user (10)
func TestIntegration_Security_ConcurrentSessionPruning(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_sec_prune_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	srv, _ := setupTestServer(conn)

	argonHash, _ := auth.HashPassword("PrunePass123!")
	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-prune-01', 'prune_pilot', 'prune@agentrix.local', $1, 'player', TRUE)
	`, argonHash)
	r.NoError(err)
	_, err = conn.Db.Exec(`
		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-prune-01', 'player')
	`)
	r.NoError(err)

	// Create 12 logins sequentially
	for i := 1; i <= 12; i++ {
		loginBody, _ := json.Marshal(map[string]string{
			"username": "prune_pilot",
			"password": "PrunePass123!",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("User-Agent", fmt.Sprintf("Device-%d", i))
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)
	}

	// In DB: active sessions should not exceed maxConcurrentSessions (10)
	var activeSessions int
	err = conn.Db.QueryRow(`
		SELECT COUNT(*) FROM sessions WHERE user_id = 'u-prune-01' AND is_revoked = FALSE
	`).Scan(&activeSessions)
	r.NoError(err)
	r.LessOrEqual(activeSessions, 10, "Active sessions must never exceed maximum limit of 10")
}

// 7. Argon2id Lazy Migration: Upgrades legacy unsalted SHA-512 hashes upon authentication
func TestIntegration_Security_Argon2idLazyRehash(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_sec_rehash_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	srv, _ := setupTestServer(conn)

	// Insert user with legacy unsalted SHA-512 hash
	legacyPassword := "LegacyPasswordSecret999!"
	h := sha512.New()
	h.Write([]byte(legacyPassword))
	legacyHash := hex.EncodeToString(h.Sum(nil))

	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES
		('u-legacy-01', 'legacy_user', 'legacy@agentrix.local', $1, 'player', TRUE)
	`, legacyHash)
	r.NoError(err)
	_, err = conn.Db.Exec(`
		INSERT INTO user_roles (user_id, role_id) VALUES
		('u-legacy-01', 'player')
	`)
	r.NoError(err)

	// Verify database currently has the 128-char hex legacy hash
	var currentHash string
	err = conn.Db.QueryRow(`SELECT password FROM users WHERE id = 'u-legacy-01'`).Scan(&currentHash)
	r.NoError(err)
	r.Equal(legacyHash, currentHash)
	r.False(auth.IsArgon2idHash(currentHash), "Initial hash must not be Argon2id")

	// Perform login with the legacy password
	loginBody, _ := json.Marshal(map[string]string{
		"username": "legacy_user",
		"password": legacyPassword,
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	recLogin := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogin, reqLogin)
	r.Equal(http.StatusOK, recLogin.Code, "Login with legacy hash must succeed")

	// Query DB again: password hash must now be upgraded to Argon2id!
	var upgradedHash string
	err = conn.Db.QueryRow(`SELECT password FROM users WHERE id = 'u-legacy-01'`).Scan(&upgradedHash)
	r.NoError(err)
	r.True(auth.IsArgon2idHash(upgradedHash), "Password hash must be lazily upgraded to Argon2id")
	r.NotEqual(legacyHash, upgradedHash)

	// Login a second time: verifies the newly upgraded Argon2id hash directly
	reqLogin2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	recLogin2 := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogin2, reqLogin2)
	r.Equal(http.StatusOK, recLogin2.Code, "Login with upgraded Argon2id hash must succeed")
}

// 8. CORS and Rate Limiting
func TestIntegration_Security_CORS_And_RateLimiting(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_sec_cors_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))
	srv, _ := setupTestServer(conn)

	// Configure strict allowed origins
	srv.AllowedOrigins = []string{"http://localhost:3000", "https://app.agentrix.io"}

	// 1. Preflight from allowed origin
	reqAllowed := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	reqAllowed.Header.Set("Origin", "http://localhost:3000")
	recAllowed := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recAllowed, reqAllowed)
	r.Equal(http.StatusOK, recAllowed.Code)
	r.Equal("http://localhost:3000", recAllowed.Header().Get("Access-Control-Allow-Origin"))
	r.Equal("true", recAllowed.Header().Get("Access-Control-Allow-Credentials"))

	// 2. Preflight from untrusted origin
	reqDisallowed := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	reqDisallowed.Header.Set("Origin", "http://evil-attacker.local")
	recDisallowed := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recDisallowed, reqDisallowed)
	r.Equal(http.StatusForbidden, recDisallowed.Code, "Disallowed CORS origin must be rejected with 403")
	r.Empty(recDisallowed.Header().Get("Access-Control-Allow-Origin"))

	// 3. Rate limiting: Set small threshold (3 requests)
	srv.RateLimiter = server.NewRateLimiter(3, 10*time.Second)

	badReqBody, _ := json.Marshal(map[string]string{
		"username": "nonexistent",
		"password": "wrongpassword",
	})

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badReqBody))
		req.RemoteAddr = "192.168.100.50:12345"
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.NotEqual(http.StatusTooManyRequests, rec.Code, "Request %d within rate limit should not be 429", i)
	}

	// 4th request must be throttled (429 Too Many Requests)
	reqThrottled := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badReqBody))
	reqThrottled.RemoteAddr = "192.168.100.50:12345"
	recThrottled := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recThrottled, reqThrottled)
	r.Equal(http.StatusTooManyRequests, recThrottled.Code, "Exceeding rate limit must return 429 Too Many Requests")
}
