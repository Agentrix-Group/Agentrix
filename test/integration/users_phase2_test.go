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
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestPhase2_CanonicalUserAndAgentAttribution(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_test_phase2_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Run canonical migrations
	r.NoError(database.Migrate(conn.Db))
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.GreaterOrEqual(ver, int64(1))

	// Verify schema changes: table users exists, participants does not exist
	var usersTableExists, participantsTableExists bool
	_ = conn.Db.QueryRow(`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name='users')`).Scan(&usersTableExists)
	_ = conn.Db.QueryRow(`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name='participants')`).Scan(&participantsTableExists)
	r.True(usersTableExists, "Table users must exist")
	r.False(participantsTableExists, "Table participants must have been renamed to users")

	// Verify column owner_user_id in agents
	var hasOwnerUserId bool
	_ = conn.Db.QueryRow(`SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_name='agents' AND column_name='owner_user_id')`).Scan(&hasOwnerUserId)
	r.True(hasOwnerUserId, "Column agents.owner_user_id must exist")

	// Insert baseline role and starfighter game
	_, err = conn.Db.Exec(`
		INSERT INTO roles (id, description) VALUES ('admin', 'System Admin'), ('participant', 'Player') ON CONFLICT DO NOTHING;
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Active game') ON CONFLICT DO NOTHING;
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)

	// 2. Register user via POST /api/v1/auth/register
	regBody, _ := json.Marshal(map[string]string{
		"username": "pilot_bob",
		"email":    "bob@agentrix.local",
		"password": "supersecretpassword123",
	})
	reqReg := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	recReg := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recReg, reqReg)
	r.Equal(http.StatusCreated, recReg.Code)

	// Verify row created in users table
	var bobID string
	err = conn.Db.QueryRow("SELECT id FROM users WHERE username = 'pilot_bob'").Scan(&bobID)
	r.NoError(err)
	r.NotEmpty(bobID)

	// 3. Login via POST /api/v1/auth/login
	loginBody, _ := json.Marshal(map[string]string{
		"username": "pilot_bob",
		"password": "supersecretpassword123",
	})
	reqLogin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	reqLogin.Header.Set("Content-Type", "application/json")
	recLogin := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recLogin, reqLogin)
	r.Equal(http.StatusOK, recLogin.Code)

	var loginResp struct {
		Token       *model.Token `json:"token"`
		User        *model.User  `json:"user"`
		Participant *model.User  `json:"participant"`
	}
	r.NoError(json.Unmarshal(recLogin.Body.Bytes(), &loginResp))
	r.NotNil(loginResp.Token)
	r.NotNil(loginResp.User)
	r.Equal("pilot_bob", loginResp.User.Username)
	r.Equal(bobID, loginResp.User.Id)

	// MUST NEVER leak password hash in response
	r.NotContains(recLogin.Body.String(), "supersecretpassword123")
	r.NotContains(recLogin.Body.String(), "password")

	// Validate JWT claims
	claims, err := authMod.ValidateAccessToken(loginResp.Token.AccessToken)
	r.NoError(err)
	r.Equal(bobID, claims.UserID)
	r.Equal(bobID, claims.Subject)
	r.Equal(bobID, claims.ParticipantId)

	// 4. Create an agent as Bob (no owner_user_id passed in body; must auto-bind to Bob)
	createAgentBody, _ := json.Marshal(map[string]interface{}{
		"id":          "agent-bob-viper",
		"name":        "BobViper",
		"game_id":     "starfighter",
		"description": "Starfighter bot",
	})
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader(createAgentBody))
	reqCreate.Header.Set("Authorization", fmt.Sprintf("Bearer %s", loginResp.Token.AccessToken))
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recCreate, reqCreate)
	r.Equal(http.StatusCreated, recCreate.Code)

	// Verify DB record has owner_user_id = Bob's ID
	var dbOwnerID string
	err = conn.Db.QueryRow("SELECT owner_user_id FROM agents WHERE id = 'agent-bob-viper'").Scan(&dbOwnerID)
	r.NoError(err)
	r.Equal(bobID, dbOwnerID)

	// 5. Query /api/v1/me/agents as Bob
	reqMyAgents := httptest.NewRequest(http.MethodGet, "/api/v1/me/agents", nil)
	reqMyAgents.Header.Set("Authorization", fmt.Sprintf("Bearer %s", loginResp.Token.AccessToken))
	recMyAgents := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recMyAgents, reqMyAgents)
	r.Equal(http.StatusOK, recMyAgents.Code)

	var myAgents []model.Agent
	r.NoError(json.Unmarshal(recMyAgents.Body.Bytes(), &myAgents))
	r.Len(myAgents, 1)
	r.Equal("BobViper", myAgents[0].Name)
	r.Equal(bobID, myAgents[0].OwnerUserId)
	r.Equal(bobID, myAgents[0].ParticipantId)

	// 6. Security authorization test:
	// Bob attempts to create an agent for another user (Alice)
	reqSpoof := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader([]byte(`{
		"name": "SpoofedBot",
		"game_id": "starfighter",
		"owner_user_id": "other-user-uuid"
	}`)))
	reqSpoof.Header.Set("Authorization", fmt.Sprintf("Bearer %s", loginResp.Token.AccessToken))
	reqSpoof.Header.Set("Content-Type", "application/json")
	recSpoof := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recSpoof, reqSpoof)
	r.Equal(http.StatusForbidden, recSpoof.Code, "Non-admin cannot create agent with different owner_user_id")

	// 7. Admin CAN create agent for another user
	_, err = conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id) VALUES ('admin-uuid', 'admin_boss', 'boss@agentrix.local', 'hash', 'admin') ON CONFLICT DO NOTHING;
	`)
	r.NoError(err)

	adminToken, err := authMod.GenerateAuthToken("admin-uuid", common.RoleAdmin)
	r.NoError(err)
	reqAdminCreate := httptest.NewRequest(http.MethodPost, "/api/v1/agents", bytes.NewReader([]byte(fmt.Sprintf(`{
		"id": "agent-for-bob-by-admin",
		"name": "GiftFromAdmin",
		"game_id": "starfighter",
		"owner_user_id": "%s"
	}`, bobID))))
	reqAdminCreate.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken.AccessToken))
	reqAdminCreate.Header.Set("Content-Type", "application/json")
	recAdminCreate := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recAdminCreate, reqAdminCreate)
	r.Equal(http.StatusCreated, recAdminCreate.Code)

	_ = ctx
}
