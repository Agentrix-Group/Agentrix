package integration

import (
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

func TestIntegration_Rankings_DeterministicRecalculationAndTiebreakers(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_rankings_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))

	// Baseline game and users
	_, err := conn.Db.Exec(`
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Active game') ON CONFLICT DO NOTHING;
		INSERT INTO users (id, username, email, password, role_id, active) VALUES 
		('u-adm', 'admin_rank', 'adm@agentrix.local', 'hash', 'admin', TRUE),
		('u-p1', 'pilot_a', 'a@agentrix.local', 'hash', 'player', TRUE),
		('u-p2', 'pilot_b', 'b@agentrix.local', 'hash', 'player', TRUE),
		('u-p3', 'pilot_c', 'c@agentrix.local', 'hash', 'player', TRUE),
		('u-p4', 'pilot_d', 'd@agentrix.local', 'hash', 'player', TRUE);
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)

	// Create 4 agents and submissions
	agents := []string{"agent-a", "agent-b", "agent-c", "agent-d"}
	users := []string{"u-p1", "u-p2", "u-p3", "u-p4"}
	subs := []string{"sub-a", "sub-b", "sub-c", "sub-d"}

	for i := 0; i < 4; i++ {
		_, err = conn.Db.Exec(`
			INSERT INTO agents (id, name, description, owner_user_id, game_id, active) 
			VALUES ($1, $2, 'Test bot', $3, 'starfighter', TRUE)
		`, agents[i], "Bot "+agents[i], users[i])
		r.NoError(err)

		_, err = conn.Db.Exec(`
			INSERT INTO submissions (id, agent_id, version, code_path, status, active)
			VALUES ($1, $2, 1, '/code.py', 'validated', TRUE)
		`, subs[i], agents[i])
		r.NoError(err)
	}

	// Create contest with explicit ScoringPolicy
	contestId := "contest-championship"
	policyJSON := `{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}`
	_, err = conn.Db.Exec(`
		INSERT INTO contests (id, name, description, status, scoring_policy)
		VALUES ($1, 'Tournament Championship', 'Deterministic leaderboard test', 'active', $2::jsonb)
	`, contestId, policyJSON)
	r.NoError(err)

	// Enroll all 4 agents
	for i := 0; i < 4; i++ {
		_, err = conn.Db.Exec(`
			INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status)
			VALUES ($1, $2, $3, $4, $5, 'active')
		`, fmt.Sprintf("entry-%d", i+1), contestId, agents[i], users[i], subs[i])
		r.NoError(err)
	}

	// Helper to commit a match with two participants
	commitMatch := func(matchId, runId string, sub1, sub2 string, score1, score2, rank1, rank2 int) {
		tFin := time.Now().UTC()
		_, err := conn.Db.Exec(`
			INSERT INTO matches (id, contest_id, game_id, status, active, finished_at)
			VALUES ($1, $2, 'starfighter', 'finished', TRUE, $3)
		`, matchId, contestId, tFin)
		r.NoError(err)

		_, err = conn.Db.Exec(`
			INSERT INTO match_jobs (id, match_id, status, fencing_token)
			VALUES ($1, $1, 'completed', 100)
		`, matchId)
		r.NoError(err)

		_, err = conn.Db.Exec(`
			INSERT INTO match_runs (id, match_id, worker_id, fencing_token, status, finished_at)
			VALUES ($1, $2, 'worker-1', 100, 'committed', $3)
		`, runId, matchId, tFin)
		r.NoError(err)

		_, err = conn.Db.Exec(`
			UPDATE matches SET committed_run_id = $1 WHERE id = $2
		`, runId, matchId)
		r.NoError(err)

		_, err = conn.Db.Exec(`
			INSERT INTO results (id, match_id, match_run_id, submission_id, score, "rank", status, created_at)
			VALUES 
			($1, $2, $3, $4, $5, $6, 'finished', $7),
			($8, $2, $3, $9, $10, $11, 'finished', $7)
		`, matchId+"-r1", matchId, runId, sub1, score1, rank1, tFin, matchId+"-r2", sub2, score2, rank2)
		r.NoError(err)
	}

	// Round-Robin Matches:
	// M1: A (100) vs B (50) -> A wins
	commitMatch("m1", "run-m1", subs[0], subs[1], 100, 50, 1, 2)
	// M2: C (80) vs D (40) -> C wins
	commitMatch("m2", "run-m2", subs[2], subs[3], 80, 40, 1, 2)
	// M3: A (90) vs C (60) -> A wins
	commitMatch("m3", "run-m3", subs[0], subs[2], 90, 60, 1, 2)
	// M4: B (70) vs D (50) -> B wins
	commitMatch("m4", "run-m4", subs[1], subs[3], 70, 50, 1, 2)
	// M5: A (120) vs D (30) -> A wins
	commitMatch("m5", "run-m5", subs[0], subs[3], 120, 30, 1, 2)
	// M6: B (85) vs C (80) -> B wins
	commitMatch("m6", "run-m6", subs[1], subs[2], 85, 80, 1, 2)

	// Execute full recalculation
	rankings, err := srv.Service.RecalculateContestRankings(ctx, contestId)
	r.NoError(err)
	r.Len(rankings, 4)

	// Verified Leaderboard:
	// 1. Agent A: 3 matches, 3 wins, 0 losses, 9 points
	r.Equal("agent-a", rankings[0].AgentId)
	r.Equal(1, rankings[0].Rank)
	r.Equal(9, rankings[0].Points)
	r.Equal(3, rankings[0].Wins)
	r.Equal(0, rankings[0].Losses)

	// 2. Agent B: 3 matches, 2 wins, 1 loss, 6 points
	r.Equal("agent-b", rankings[1].AgentId)
	r.Equal(2, rankings[1].Rank)
	r.Equal(6, rankings[1].Points)
	r.Equal(2, rankings[1].Wins)
	r.Equal(1, rankings[1].Losses)

	// 3. Agent C: 3 matches, 1 win, 2 losses, 3 points
	r.Equal("agent-c", rankings[2].AgentId)
	r.Equal(3, rankings[2].Rank)
	r.Equal(3, rankings[2].Points)
	r.Equal(1, rankings[2].Wins)
	r.Equal(2, rankings[2].Losses)

	// 4. Agent D: 3 matches, 0 wins, 3 losses, 0 points
	r.Equal("agent-d", rankings[3].AgentId)
	r.Equal(4, rankings[3].Rank)
	r.Equal(0, rankings[3].Points)
	r.Equal(0, rankings[3].Wins)
	r.Equal(3, rankings[3].Losses)

	// Re-calculating must NOT alter any result
	recalculated, err := srv.Service.RecalculateContestRankings(ctx, contestId)
	r.NoError(err)
	for i := range rankings {
		r.Equal(rankings[i].AgentId, recalculated[i].AgentId)
		r.Equal(rankings[i].Rank, recalculated[i].Rank)
		r.Equal(rankings[i].Points, recalculated[i].Points)
		r.Equal(rankings[i].Score, recalculated[i].Score)
		r.Equal(rankings[i].TiebreakerScore, recalculated[i].TiebreakerScore)
	}

	// Test Snapshot Publication & Spectator Access
	adminTokens, err := authMod.GenerateAuthToken("u-adm", "admin")
	r.NoError(err)

	// Publish snapshot
	publishReq := httptest.NewRequest(http.MethodPost, "/api/v1/contests/"+contestId+"/rankings/publish", nil)
	publishReq.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	publishRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(publishRec, publishReq)
	r.Equal(http.StatusOK, publishRec.Code)

	var snapResp model.RankingSnapshot
	r.NoError(json.NewDecoder(publishRec.Body).Decode(&snapResp))
	r.Equal(1, snapResp.Version)
	r.Equal(contestId, snapResp.ContestId)
	r.Len(snapResp.Rankings, 4)

	// Spectator access without Authorization header
	getSnapReq := httptest.NewRequest(http.MethodGet, "/api/v1/contests/"+contestId+"/rankings/snapshots/1", nil)
	getSnapRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(getSnapRec, getSnapReq)
	r.Equal(http.StatusOK, getSnapRec.Code)

	var spectatorSnap model.RankingSnapshot
	r.NoError(json.NewDecoder(getSnapRec.Body).Decode(&spectatorSnap))
	r.Equal(1, spectatorSnap.Version)
	r.Equal("agent-a", spectatorSnap.Rankings[0].AgentId)

	// List snapshots public endpoint
	listSnapReq := httptest.NewRequest(http.MethodGet, "/api/v1/contests/"+contestId+"/rankings/snapshots", nil)
	listSnapRec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(listSnapRec, listSnapReq)
	r.Equal(http.StatusOK, listSnapRec.Code)
}

func TestIntegration_Rankings_CascadingTiebreakers(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_tiebreakers_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))

	_, err := conn.Db.Exec(`
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Active game') ON CONFLICT DO NOTHING;
		INSERT INTO users (id, username, email, password, role_id, active) VALUES 
		('u-1', 'user1', 'u1@agentrix.local', 'hash', 'player', TRUE),
		('u-2', 'user2', 'u2@agentrix.local', 'hash', 'player', TRUE);
		INSERT INTO agents (id, name, description, owner_user_id, game_id, active) VALUES 
		('agent-x', 'Agent X', 'Bot X', 'u-1', 'starfighter', TRUE),
		('agent-y', 'Agent Y', 'Bot Y', 'u-2', 'starfighter', TRUE);
		INSERT INTO submissions (id, agent_id, version, code_path, status, active) VALUES 
		('sub-x', 'agent-x', 1, '/code.py', 'validated', TRUE),
		('sub-y', 'agent-y', 1, '/code.py', 'validated', TRUE);
	`)
	r.NoError(err)

	srv, _ := setupTestServer(conn)

	contestId := "contest-tiebreaker"
	policyJSON := `{"win_points":3,"draw_points":1,"loss_points":0,"disqualification_penalty":0,"tiebreakers":["score_diff","head_to_head","wins"]}`
	_, err = conn.Db.Exec(`
		INSERT INTO contests (id, name, description, status, scoring_policy)
		VALUES ($1, 'Tiebreaker Contest', 'Testing tiebreakers', 'active', $2::jsonb)
	`, contestId, policyJSON)
	r.NoError(err)

	_, err = conn.Db.Exec(`
		INSERT INTO contest_entries (id, contest_id, agent_id, user_id, submission_id, status) VALUES 
		('e-x', $1, 'agent-x', 'u-1', 'sub-x', 'active'),
		('e-y', $1, 'agent-y', 'u-2', 'sub-y', 'active')
	`, contestId)
	r.NoError(err)

	tFin1 := time.Now().UTC().Add(-time.Hour)
	tFin2 := time.Now().UTC()

	// Match 1: X beats Y (X: 100, Y: 80) -> X gets 3 points, score_diff +20
	_, err = conn.Db.Exec(`
		INSERT INTO matches (id, contest_id, game_id, status, active, finished_at)
		VALUES ('m-tb-1', $1, 'starfighter', 'finished', TRUE, $2)
	`, contestId, tFin1)
	r.NoError(err)

	_, err = conn.Db.Exec(`INSERT INTO match_jobs (id, match_id, status, fencing_token) VALUES ('m-tb-1', 'm-tb-1', 'completed', 1)`)
	r.NoError(err)

	_, err = conn.Db.Exec(`INSERT INTO match_runs (id, match_id, worker_id, fencing_token, status, finished_at) VALUES ('run-tb-1', 'm-tb-1', 'w1', 1, 'committed', $1)`, tFin1)
	r.NoError(err)

	_, err = conn.Db.Exec(`UPDATE matches SET committed_run_id = 'run-tb-1' WHERE id = 'm-tb-1'`)
	r.NoError(err)

	_, err = conn.Db.Exec(`
		INSERT INTO results (id, match_id, match_run_id, submission_id, score, "rank", status, created_at) VALUES 
		('r-tb-1-x', 'm-tb-1', 'run-tb-1', 'sub-x', 100, 1, 'finished', $1),
		('r-tb-1-y', 'm-tb-1', 'run-tb-1', 'sub-y', 80, 2, 'finished', $1)
	`, tFin1)
	r.NoError(err)

	// Match 2: Y beats X (Y: 50, X: 40) -> Y gets 3 points, score_diff +10
	_, err = conn.Db.Exec(`
		INSERT INTO matches (id, contest_id, game_id, status, active, finished_at)
		VALUES ('m-tb-2', $1, 'starfighter', 'finished', TRUE, $2)
	`, contestId, tFin2)
	r.NoError(err)

	_, err = conn.Db.Exec(`INSERT INTO match_jobs (id, match_id, status, fencing_token) VALUES ('m-tb-2', 'm-tb-2', 'completed', 2)`)
	r.NoError(err)

	_, err = conn.Db.Exec(`INSERT INTO match_runs (id, match_id, worker_id, fencing_token, status, finished_at) VALUES ('run-tb-2', 'm-tb-2', 'w1', 2, 'committed', $1)`, tFin2)
	r.NoError(err)

	_, err = conn.Db.Exec(`UPDATE matches SET committed_run_id = 'run-tb-2' WHERE id = 'm-tb-2'`)
	r.NoError(err)

	_, err = conn.Db.Exec(`
		INSERT INTO results (id, match_id, match_run_id, submission_id, score, "rank", status, created_at) VALUES 
		('r-tb-2-y', 'm-tb-2', 'run-tb-2', 'sub-y', 50, 1, 'finished', $1),
		('r-tb-2-x', 'm-tb-2', 'run-tb-2', 'sub-x', 40, 2, 'finished', $1)
	`, tFin2)
	r.NoError(err)

	// Both have: 1 win, 1 loss -> 3 points each.
	// But X's net score_diff = (100 - 80) + (40 - 50) = +20 - 10 = +10.
	// Y's net score_diff = (80 - 100) + (50 - 40) = -20 + 10 = -10.
	// ScoreDiff tiebreaker puts X at Rank 1 and Y at Rank 2!
	rankings, err := srv.Service.RecalculateContestRankings(ctx, contestId)
	r.NoError(err)
	r.Len(rankings, 2)

	r.Equal("agent-x", rankings[0].AgentId)
	r.Equal(1, rankings[0].Rank)
	r.Equal(3, rankings[0].Points)
	r.Equal(float64(10), rankings[0].TiebreakerScore)

	r.Equal("agent-y", rankings[1].AgentId)
	r.Equal(2, rankings[1].Rank)
	r.Equal(3, rankings[1].Points)
	r.Equal(float64(-10), rankings[1].TiebreakerScore)
}

func TestIntegration_Rankings_IdempotencyAndCommittedRuns(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_idem_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))

	_, err := conn.Db.Exec(`
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Active game') ON CONFLICT DO NOTHING;
		INSERT INTO users (id, username, email, password, role_id, active) VALUES 
		('u-1', 'user1', 'u1@agentrix.local', 'hash', 'player', TRUE),
		('u-2', 'user2', 'u2@agentrix.local', 'hash', 'player', TRUE);
		INSERT INTO agents (id, name, description, owner_user_id, game_id, active) VALUES 
		('a-1', 'A 1', 'Bot', 'u-1', 'starfighter', TRUE),
		('a-2', 'A 2', 'Bot', 'u-2', 'starfighter', TRUE);
		INSERT INTO submissions (id, agent_id, version, code_path, status, active) VALUES 
		('sub-1', 'a-1', 1, '/code.py', 'validated', TRUE),
		('sub-2', 'a-2', 1, '/code.py', 'validated', TRUE);
		INSERT INTO contests (id, name, description, status) VALUES ('c-test', 'Contest Test', 'Test', 'active');
	`)
	r.NoError(err)

	srv, _ := setupTestServer(conn)

	// 1. Schedule a match without committed run (status = 'scheduled')
	_, err = conn.Db.Exec(`
		INSERT INTO matches (id, contest_id, game_id, status, active)
		VALUES ('m-uncommitted', 'c-test', 'starfighter', 'scheduled', TRUE)
	`)
	r.NoError(err)

	// Apply incremental should be a no-op for uncommitted match
	err = srv.Service.ApplyMatchResultIncremental(ctx, "m-uncommitted")
	r.NoError(err)

	ranks, err := srv.Service.ListRankingsByContest(ctx, "c-test")
	r.NoError(err)
	r.Empty(ranks)

	// 2. Now finish match and commit run
	now := time.Now().UTC()
	_, err = conn.Db.Exec(`INSERT INTO match_jobs (id, match_id, status, fencing_token) VALUES ('m-uncommitted', 'm-uncommitted', 'completed', 1)`)
	r.NoError(err)

	_, err = conn.Db.Exec(`INSERT INTO match_runs (id, match_id, worker_id, fencing_token, status, finished_at) VALUES ('run-c1', 'm-uncommitted', 'w1', 1, 'committed', $1)`, now)
	r.NoError(err)

	_, err = conn.Db.Exec(`
		UPDATE matches SET status = 'finished', committed_run_id = 'run-c1', finished_at = $1 WHERE id = 'm-uncommitted'
	`, now)
	r.NoError(err)

	_, err = conn.Db.Exec(`
		INSERT INTO results (id, match_id, match_run_id, submission_id, score, "rank", status, created_at) VALUES 
		('r-1', 'm-uncommitted', 'run-c1', 'sub-1', 100, 1, 'finished', $1),
		('r-2', 'm-uncommitted', 'run-c1', 'sub-2', 50, 2, 'finished', $1)
	`, now)
	r.NoError(err)

	// First incremental application
	err = srv.Service.ApplyMatchResultIncremental(ctx, "m-uncommitted")
	r.NoError(err)

	ranks1, err := srv.Service.ListRankingsByContest(ctx, "c-test")
	r.NoError(err)
	r.Len(ranks1, 2)
	r.Equal("a-1", ranks1[0].AgentId)
	r.Equal(3, ranks1[0].Points)

	// Second incremental application: must be completely idempotent
	err = srv.Service.ApplyMatchResultIncremental(ctx, "m-uncommitted")
	r.NoError(err)

	ranks2, err := srv.Service.ListRankingsByContest(ctx, "c-test")
	r.NoError(err)
	r.Len(ranks2, 2)
	r.Equal(ranks1[0].Points, ranks2[0].Points) // Points are not doubled!
	r.Equal(ranks1[0].Score, ranks2[0].Score)
}

func TestIntegration_Rankings_RecalculateEndpointPermissions(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_rank_perm_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	r.NoError(database.Migrate(conn.Db))

	_, err := conn.Db.Exec(`
		INSERT INTO users (id, username, email, password, role_id, active) VALUES 
		('u-adm', 'admin1', 'adm@agentrix.local', 'hash', 'admin', TRUE),
		('u-org', 'organizer1', 'org@agentrix.local', 'hash', 'organizer', TRUE),
		('u-ply', 'player1', 'ply@agentrix.local', 'hash', 'player', TRUE);
		INSERT INTO contests (id, name, description, status) VALUES ('c-perm', 'Contest Perm', 'Test', 'active');
	`)
	r.NoError(err)

	srv, authMod := setupTestServer(conn)

	playerTokens, err := authMod.GenerateAuthToken("u-ply", "player")
	r.NoError(err)
	organizerTokens, err := authMod.GenerateAuthToken("u-org", "organizer")
	r.NoError(err)
	adminTokens, err := authMod.GenerateAuthToken("u-adm", "admin")
	r.NoError(err)

	// 1. Anonymous request to recalculate -> 401 Unauthorized
	reqAnon := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-perm/rankings/recalculate", nil)
	recAnon := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recAnon, reqAnon)
	r.Equal(http.StatusUnauthorized, recAnon.Code)

	// 2. Player request to recalculate -> 403 Forbidden (requires contests:manage)
	reqPlayer := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-perm/rankings/recalculate", nil)
	reqPlayer.Header.Set("Authorization", "Bearer "+playerTokens.AccessToken)
	recPlayer := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recPlayer, reqPlayer)
	r.Equal(http.StatusForbidden, recPlayer.Code)

	// 3. Organizer request to recalculate -> 200 OK (organizer has contests:manage)
	reqOrg := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-perm/rankings/recalculate", nil)
	reqOrg.Header.Set("Authorization", "Bearer "+organizerTokens.AccessToken)
	recOrg := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recOrg, reqOrg)
	r.Equal(http.StatusOK, recOrg.Code)

	// 4. Admin request to publish -> 200 OK (admin has rankings:publish)
	reqAdmin := httptest.NewRequest(http.MethodPost, "/api/v1/contests/c-perm/rankings/publish", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+adminTokens.AccessToken)
	recAdmin := httptest.NewRecorder()
	srv.Handler.ServeHTTP(recAdmin, reqAdmin)
	r.Equal(http.StatusOK, recAdmin.Code)
}
