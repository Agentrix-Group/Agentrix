package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Demo_FullSmokeCycle(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_smoke_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Run migrations to schema version 4
	r.NoError(database.Migrate(conn.Db))
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.GreaterOrEqual(ver, int64(4))

	// 2. Apply SQL seeds
	seedSQL, err := os.ReadFile("../../script/data/00_seeds_postgresql.sql")
	r.NoError(err)
	_, err = conn.Db.Exec(string(seedSQL))
	r.NoError(err)

	// Setup server and auth module
	srv, _ := setupTestServer(conn)

	// Ensure Starfighter engine and game manifest are loaded
	gamesDir, _ := filepath.Abs("../../games")
	r.NoError(game.GetRegistry().LoadGamesFromDir(gamesDir))

	manifest := game.GetRegistry().GetManifest("starfighter")
	r.NotNil(manifest)
	manifest.BinaryPath = resolveTestPath("bin/starfighter-engine")

	// --- Check 1: Health & Readiness probe ---
	{
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var readiness map[string]any
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &readiness))
		r.Equal("READY", readiness["status"])
	}

	// --- Check 2: Login Admin ---
	var adminToken string
	{
		body, _ := json.Marshal(model.LoginRequest{
			Username: "admin",
			Password: "admin123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var resp model.LoginResponse
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		r.NotNil(resp.Token)
		r.Equal("admin", resp.User.RoleId)
		adminToken = resp.Token.AccessToken
	}

	// --- Check 3: Login Participante (pilot_alpha) ---
	var pilotToken string
	{
		body, _ := json.Marshal(model.LoginRequest{
			Username: "pilot_alpha",
			Password: "pilot123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var resp model.LoginResponse
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		r.NotNil(resp.Token)
		r.Equal("player", resp.User.RoleId)
		pilotToken = resp.Token.AccessToken
	}

	// --- Check 4: Intento fallido de participante accediendo a /api/v1/admin/* (debe retornar 403) ---
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit", nil)
		req.Header.Set("Authorization", "Bearer "+pilotToken)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusForbidden, rec.Code, "Non-admin user accessing /api/v1/admin/* must receive 403 Forbidden")
	}

	// --- Check 5: Subida/creación de agente y submission lista ---
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/me/agents", nil)
		req.Header.Set("Authorization", "Bearer "+pilotToken)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var agents []model.Agent
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &agents))
		r.NotEmpty(agents)
		r.Equal("agent-star-hunter", agents[0].Id)
	}

	botHunterPath := resolveTestPath("games/starfighter/examples/bot_hunter.py")
	botEvasivePath := resolveTestPath("games/starfighter/examples/bot_evasive.py")
	r.FileExists(botHunterPath)
	r.FileExists(botEvasivePath)

	// Seed real submission code paths into DB
	_, err = conn.Db.Exec(`UPDATE submissions SET code_path = $1, status = 'ready', active = TRUE WHERE id = 'sub-star-hunter-1'`, botHunterPath)
	r.NoError(err)
	_, err = conn.Db.Exec(`UPDATE submissions SET code_path = $1, status = 'ready', active = TRUE WHERE id = 'sub-star-evasive-1'`, botEvasivePath)
	r.NoError(err)

	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/submissions", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var subs []model.Submission
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &subs))
		r.GreaterOrEqual(len(subs), 2)
	}

	// --- Check 6: Inscripción en concurso ---
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests/starfighter-cup-2026/entries", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var entries []model.ContestEntry
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &entries))
		r.Len(entries, 2)
	}

	// --- Check 7: Programación de partida real ---
	matchID := "match-smoke-001"
	runID := "run-smoke-001"
	contestID := "starfighter-cup-2026"

	_, err = conn.Db.Exec(`
		INSERT INTO matches (id, contest_id, game_id, status, seed, active, created_at)
		VALUES ($1, $2, 'starfighter', 'scheduled', 42, TRUE, NOW())
	`, matchID, contestID)
	r.NoError(err)

	_, err = conn.Db.Exec(`
		INSERT INTO match_slots (id, match_id, slot_index, contest_entry_id, submission_id, agent_name, username, created_at)
		VALUES
		('slot-sm-1', $1, 0, 'entry-hunter-001', 'sub-star-hunter-1', 'StarHunter', 'pilot_alpha', NOW()),
		('slot-sm-2', $1, 1, 'entry-evasive-001', 'sub-star-evasive-1', 'StarEvasive', 'pilot_beta', NOW())
	`, matchID)
	r.NoError(err)

	_, err = conn.Db.Exec(`
		INSERT INTO match_jobs (id, match_id, contest_id, game_id, status, fencing_token, available_at)
		VALUES ('job-sm-001', $1, $2, 'starfighter', 'reserved', 1, NOW())
	`, matchID, contestID)
	r.NoError(err)

	// --- Check 8: Ejecución de partida real con simulación y bots ---
	sandbox := executor.NewSandbox(0)
	matchExec := executor.NewMatchExecutor(srv.Service, sandbox)

	job := &connection.MatchJob{
		JobId:         "job-sm-001",
		RunId:         runID,
		MatchId:       matchID,
		ContestId:     contestID,
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-star-hunter-1", "sub-star-evasive-1"},
		Seed:          42,
		FencingToken:  1,
	}

	execErr := matchExec.Execute(ctx, job)
	r.NoError(execErr, "Match execution with real Bevy/Rapier engine and bots must succeed")

	// --- Check 9: Espera terminal con timeout ---
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/matches/"+matchID, nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var match model.Match
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &match))
		r.Equal("finished", match.Status)
		r.NotEmpty(match.ReplayId)
		r.NotNil(match.FinishedAt)
		r.NotNil(match.CommittedRunId)
		r.Equal(runID, *match.CommittedRunId)
	}

	// --- Check 10: Resultados normalizados ---
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/results", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var results []model.Result
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &results))
		r.GreaterOrEqual(len(results), 2)
	}

	// --- Check 11: Replay parseable y visible ---
	{
		match, err := srv.Service.GetMatch(ctx, matchID)
		r.NoError(err)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/replays/"+match.ReplayId, nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var rep model.Replay
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &rep))
		r.Equal(matchID, rep.MatchId)
		r.Greater(rep.DurationTicks, 0)
		r.NotEmpty(rep.Sha256)
	}

	// --- Check 12: Ranking actualizado determinista ---
	{
		rankings, err := srv.Service.RecalculateContestRankings(ctx, contestID)
		r.NoError(err)
		r.Len(rankings, 2)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests/"+contestID+"/rankings", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var pubRanks []model.Ranking
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &pubRanks))
		r.Len(pubRanks, 2)
		r.Equal(1, pubRanks[0].Rank)
		r.Equal(2, pubRanks[1].Rank)
	}

	// --- Check 13: Ausencia de jobs/runs huérfanos ---
	{
		var pendingOrReservedJobs int
		err := conn.Db.QueryRow("SELECT COUNT(*) FROM match_jobs WHERE status IN ('reserved', 'running')").Scan(&pendingOrReservedJobs)
		r.NoError(err)
		r.Equal(0, pendingOrReservedJobs, "No jobs should remain hanging in reserved or running status")

		var runningRuns int
		err = conn.Db.QueryRow("SELECT COUNT(*) FROM match_runs WHERE status = 'running'").Scan(&runningRuns)
		r.NoError(err)
		r.Equal(0, runningRuns, "No runs should remain in running state")
	}
}
