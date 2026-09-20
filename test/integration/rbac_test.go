package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/common"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

func TestIntegration_RBAC_DynamicEnforcement(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_rbac_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Migrate canonical database schema
	r.NoError(database.Migrate(conn.Db))
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.GreaterOrEqual(ver, int64(1))

	// 2. Insert test game
	_, err = conn.Db.Exec(`
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Active game') ON CONFLICT DO NOTHING;
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)

	// Register player and admin users in DB
	_, err = conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES 
		('u-player-01', 'pilot_charlie', 'charlie@agentrix.local', 'hash', 'player', TRUE),
		('u-spectator-01', 'viewer_dan', 'dan@agentrix.local', 'hash', 'spectator', TRUE),
		('u-admin-01', 'commander_eve', 'eve@agentrix.local', 'hash', 'admin', TRUE),
		('u-inactive-01', 'banned_fred', 'fred@agentrix.local', 'hash', 'player', FALSE);
	`)
	r.NoError(err)

	playerTokens, err := authMod.GenerateAuthToken("u-player-01", "player")
	r.NoError(err)
	spectatorTokens, err := authMod.GenerateAuthToken("u-spectator-01", "spectator")
	r.NoError(err)
	adminTokens, err := authMod.GenerateAuthToken("u-admin-01", common.RoleAdmin)
	r.NoError(err)
	inactiveTokens, err := authMod.GenerateAuthToken("u-inactive-01", "player")
	r.NoError(err)

	// Test 1: Player CAN create their own agent (has submit-agent permission)
	{
		agentBody, _ := json.Marshal(map[string]interface{}{
			"id":      "agent-charlie-1",
			"name":    "CharlieJet",
			"game_id": "starfighter",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(agentBody))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", playerTokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusCreated, rec.Code, "Player must be allowed to create agent")
	}

	// Test 2: Spectator CANNOT create agent (lacks submit-agent permission) -> 403
	{
		agentBody, _ := json.Marshal(map[string]interface{}{
			"id":      "agent-dan-1",
			"name":    "DanSpectatorBot",
			"game_id": "starfighter",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(agentBody))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", spectatorTokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusForbidden, rec.Code, "Spectator must be denied agent creation with 403 Forbidden")
	}

	// Test 3: Player CANNOT invoke admin endpoint (POST /api/v1/contests) -> 403
	{
		contestBody, _ := json.Marshal(map[string]interface{}{
			"name":    "Unauthorized Contest",
			"game_id": "starfighter",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/contests", bytes.NewReader(contestBody))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", playerTokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusForbidden, rec.Code, "Player must not invoke admin-only contest creation")
	}

	// Test 4: Admin CAN invoke admin endpoint (POST /api/v1/contests) -> 201
	{
		contestBody, _ := json.Marshal(map[string]interface{}{
			"id":      "contest-eve-01",
			"name":    "Eve Tournament",
			"game_id": "starfighter",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/contests", bytes.NewReader(contestBody))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminTokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusCreated, rec.Code, "Admin must be authorized to create contest")
	}

	// Test 5: Inactive user account is rejected even with valid token
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me/agents", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", inactiveTokens.AccessToken))
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusForbidden, rec.Code, "Inactive account must be rejected")
	}

	// Test 6: Invalidation of permission cache
	service.InvalidatePermissionCache()

	// Verify permissions cache reload still allows Charlie to read agents
	reqRead := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	reqRead.Header.Set("Authorization", fmt.Sprintf("Bearer %s", playerTokens.AccessToken))
	recRead := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recRead, reqRead)
	r.Equal(http.StatusOK, recRead.Code, "Player can read agents")

	_ = ctx
}
