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

	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Contests_EntriesAndLeaderboardSeparation(t *testing.T) {
	r := require.New(t)
	_ = context.Background()

	dbName := fmt.Sprintf("agentrix_contests_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Verify Goose migration reaches canonical version 1
	r.NoError(database.Migrate(conn.Db))
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.GreaterOrEqual(ver, int64(1))

	// Verify contest_entries table exists
	var tableName string
	err = conn.Db.QueryRow(`
		SELECT table_name FROM information_schema.tables 
		WHERE table_name = 'contest_entries'
	`).Scan(&tableName)
	r.NoError(err)
	r.Equal("contest_entries", tableName)

	// 2. Setup baseline test data (Game, Users, Contests)
	_, err = conn.Db.Exec(`
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Active game') ON CONFLICT DO NOTHING;
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)

	_, err = conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES 
		('u-pilot-01', 'pilot_alice', 'alice@agentrix.local', 'hash', 'player', TRUE),
		('u-pilot-02', 'pilot_bob', 'bob@agentrix.local', 'hash', 'player', TRUE),
		('u-admin-01', 'admin_contests', 'admin@agentrix.local', 'hash', 'admin', TRUE);
	`)
	r.NoError(err)

	aliceTokens, err := authMod.GenerateAuthToken("u-pilot-01", "player")
	r.NoError(err)
	bobTokens, err := authMod.GenerateAuthToken("u-pilot-02", "player")
	r.NoError(err)

	// Create open contest and closed contest
	openContestId := "c-open-01"
	closedContestId := "c-closed-01"
	_, err = conn.Db.Exec(`
		INSERT INTO contests (id, name, description, game_id, state, active, starts_at, ends_at) VALUES 
		('c-open-01', 'Open Tournament', 'Open for enrollment', 'starfighter', 'registration_open', TRUE, NOW(), NOW() + INTERVAL '7 days'),
		('c-closed-01', 'Running Match', 'Already in progress', 'starfighter', 'in_progress', TRUE, NOW() - INTERVAL '1 hour', NOW() + INTERVAL '1 hour');
	`)
	r.NoError(err)

	// Create agents
	_, err = conn.Db.Exec(`
		INSERT INTO agents (id, name, game_id, owner_user_id, active) VALUES 
		('agent-alice-01', 'AliceFighter', 'starfighter', 'u-pilot-01', TRUE),
		('agent-bob-01', 'BobFighter', 'starfighter', 'u-pilot-02', TRUE);
	`)
	r.NoError(err)

	// Test 1: Successful enrollment of Alice's agent into open contest
	{
		enrollBody, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-alice-01",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/contests/%s/agents", openContestId), bytes.NewReader(enrollBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceTokens.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusCreated, rec.Code, "Expected 201 Created for valid enrollment, got: %s", rec.Body.String())

		var resp model.EnrollAgentResponse
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		r.NotNil(resp.Entry, "EnrollAgentResponse must include ContestEntry")
		r.NotNil(resp.Ranking, "EnrollAgentResponse must include initial Ranking")
		r.Equal("agent-alice-01", resp.Entry.AgentId)
		r.Equal("u-pilot-01", resp.Entry.UserId)
		r.Equal("enrolled", resp.Entry.Status)
		r.Equal("agent-alice-01", resp.Ranking.AgentId)

		// Verify database persistence in contest_entries
		var dbEntryId, dbAgentId, dbUserId, dbStatus string
		err = conn.Db.QueryRow(`
			SELECT id, agent_id, user_id, status FROM contest_entries 
			WHERE contest_id = $1 AND agent_id = $2
		`, openContestId, "agent-alice-01").Scan(&dbEntryId, &dbAgentId, &dbUserId, &dbStatus)
		r.NoError(err)
		r.Equal("agent-alice-01", dbAgentId)
		r.Equal("u-pilot-01", dbUserId)
		r.Equal("enrolled", dbStatus)

		// Verify database persistence in rankings
		var dbRankAgentId, dbRankUserId string
		var dbScore, dbRank int
		err = conn.Db.QueryRow(`
			SELECT agent_id, user_id, score, rank FROM rankings 
			WHERE contest_id = $1 AND agent_id = $2
		`, openContestId, "agent-alice-01").Scan(&dbRankAgentId, &dbRankUserId, &dbScore, &dbRank)
		r.NoError(err)
		r.Equal("agent-alice-01", dbRankAgentId)
		r.Equal("u-pilot-01", dbRankUserId)
		r.Equal(0, dbScore)
		r.Equal(1, dbRank)
	}

	// Test 2: List contest entries via GET /api/v1/contests/{id}/entries
	{
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/contests/%s/entries", openContestId), nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var entries []model.ContestEntry
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &entries))
		r.Len(entries, 1)
		r.Equal("agent-alice-01", entries[0].AgentId)
		r.Equal("u-pilot-01", entries[0].UserId)
		r.Equal("enrolled", entries[0].Status)
	}

	// Test 3: Duplicate enrollment attempt is rejected with 409 Conflict
	{
		enrollBody, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-alice-01",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/contests/%s/agents", openContestId), bytes.NewReader(enrollBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+aliceTokens.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusConflict, rec.Code)
	}

	// Test 4: Bob trying to enroll Alice's agent is rejected with 403 Forbidden
	{
		enrollBody, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-alice-01",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/contests/%s/agents", openContestId), bytes.NewReader(enrollBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+bobTokens.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusForbidden, rec.Code)
	}

	// Test 5: Bob trying to enroll into a contest not in registration_open state returns 400 Bad Request
	{
		enrollBody, _ := json.Marshal(model.EnrollAgentRequest{
			AgentId: "agent-bob-01",
		})
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/contests/%s/agents", closedContestId), bytes.NewReader(enrollBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+bobTokens.AccessToken)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusBadRequest, rec.Code)
	}
}
