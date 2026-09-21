package integration

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Bootstrap_StarfighterAndBundleAdmission(t *testing.T) {
	r := require.New(t)
	_ = context.Background()

	dbName := fmt.Sprintf("agentrix_bootstrap_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Run migrations to canonical schema version 1
	r.NoError(database.Migrate(conn.Db))
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.GreaterOrEqual(ver, int64(1))

	// 2. Apply SQL Seeds from script/data/00_seeds_postgresql.sql
	seedSQL, err := os.ReadFile("../../script/data/00_seeds_postgresql.sql")
	r.NoError(err)
	_, err = conn.Db.Exec(string(seedSQL))
	r.NoError(err, "Executing 00_seeds_postgresql.sql must succeed")

	// 3. Verify Seeded Game (Starfighter)
	var gameName, manifestPath string
	var gameActive bool
	err = conn.Db.QueryRow(`
		SELECT name, manifest_path, active FROM games WHERE id = 'starfighter'
	`).Scan(&gameName, &manifestPath, &gameActive)
	r.NoError(err)
	r.Equal("Starfighter Arena", gameName)
	r.True(gameActive)

	// 4. Verify Seeded Users and Authentication via API
	srv, _ := setupTestServer(conn)

	// Test login for admin
	{
		loginBody, _ := json.Marshal(model.LoginRequest{
			Username: "admin",
			Password: "admin123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code, "Admin login should succeed")

		var resp model.LoginResponse
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		r.NotNil(resp.Token)
		r.Equal("admin", resp.User.RoleId)
	}

	// Test login for pilot_alpha
	var pilotAlphaToken string
	{
		loginBody, _ := json.Marshal(model.LoginRequest{
			Username: "pilot_alpha",
			Password: "pilot123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code, "pilot_alpha login should succeed")

		var resp model.LoginResponse
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		r.NotNil(resp.Token)
		pilotAlphaToken = resp.Token.AccessToken
		r.Equal("player", resp.User.RoleId)
	}

	// Test login for pilot_beta
	var pilotBetaToken string
	{
		loginBody, _ := json.Marshal(model.LoginRequest{
			Username: "pilot_beta",
			Password: "pilot123",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code, "pilot_beta login should succeed")

		var resp model.LoginResponse
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &resp))
		r.NotNil(resp.Token)
		pilotBetaToken = resp.Token.AccessToken
	}

	// 5. Verify Seeded Contest and Contest Entries
	{
		req := httptest.NewRequest(http.MethodGet, "/api/v1/contests/starfighter-cup-2026/entries", nil)
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var entries []model.ContestEntry
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &entries))
		r.Len(entries, 2, "Contest should have 2 demo entries")

		foundHunter := false
		foundEvasive := false
		for _, e := range entries {
			if e.AgentId == "agent-star-hunter" {
				foundHunter = true
				r.Equal(model.ContestEntryStatusEnrolled, e.Status)
			}
			if e.AgentId == "agent-star-evasive" {
				foundEvasive = true
				r.Equal(model.ContestEntryStatusEnrolled, e.Status)
			}
		}
		r.True(foundHunter && foundEvasive, "Both StarHunter and StarEvasive must be enrolled")
	}

	// 6. Test Bot Submission Bundle Admission
	createBundle := func(botPy, manifestJSON []byte) []byte {
		buf := new(bytes.Buffer)
		zw := zip.NewWriter(buf)
		if botPy != nil {
			f, _ := zw.Create("bot.py")
			_, _ = f.Write(botPy)
		}
		if manifestJSON != nil {
			f, _ := zw.Create("agentrix.json")
			_, _ = f.Write(manifestJSON)
		}
		_ = zw.Close()
		return buf.Bytes()
	}

	validManifest, _ := json.Marshal(model.AgentPackageManifest{
		Name:            "StarHunterV2",
		Entrypoint:      "bot.py",
		ProtocolVersion: "1.0",
	})
	validBotPy, err := os.ReadFile("../../games/starfighter/examples/bot_random.py")
	r.NoError(err, "Must be able to read bot_random.py for bundle admission test")

	// 6a. Valid bundle upload by owner (pilot_alpha) succeeds
	{
		bundleBytes := createBundle(validBotPy, validManifest)
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("agent_id", "agent-star-hunter")
		part, _ := writer.CreateFormFile("bundle", "bot_bundle.zip")
		_, _ = part.Write(bundleBytes)
		_ = writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/upload", &body)
		req.Header.Set("Authorization", "Bearer "+pilotAlphaToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusCreated, rec.Code, "Valid bundle upload by owner should succeed: %s", rec.Body.String())

		var sub model.Submission
		r.NoError(json.Unmarshal(rec.Body.Bytes(), &sub))
		r.Equal("agent-star-hunter", sub.AgentId)
		r.Equal("ready", sub.Status)
	}

	// 6b. Unauthorized upload by pilot_beta for pilot_alpha's agent is rejected with 403 Forbidden
	{
		bundleBytes := createBundle(validBotPy, validManifest)
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("agent_id", "agent-star-hunter")
		part, _ := writer.CreateFormFile("bundle", "bot_bundle.zip")
		_, _ = part.Write(bundleBytes)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/upload", &body)
		req.Header.Set("Authorization", "Bearer "+pilotBetaToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusForbidden, rec.Code, "Non-owner upload must be rejected with 403 Forbidden")
	}

	// 6c. Invalid bundle (missing agentrix.json) is rejected with 400 Bad Request
	{
		invalidBundleBytes := createBundle(validBotPy, nil)
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("agent_id", "agent-star-hunter")
		part, _ := writer.CreateFormFile("bundle", "invalid_bundle.zip")
		_, _ = part.Write(invalidBundleBytes)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/submissions/upload", &body)
		req.Header.Set("Authorization", "Bearer "+pilotAlphaToken)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusBadRequest, rec.Code, "Missing manifest must be rejected with 400 Bad Request")
	}

	// 7. Test Idempotency: Re-executing seeds must not error or duplicate rows
	_, err = conn.Db.Exec(string(seedSQL))
	r.NoError(err, "Re-executing seeds must be idempotent and succeed")

	var userCount, agentCount, entryCount int
	r.NoError(conn.Db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount))
	r.NoError(conn.Db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&agentCount))
	r.NoError(conn.Db.QueryRow("SELECT COUNT(*) FROM contest_entries").Scan(&entryCount))
	r.Equal(3, userCount, "User count must remain exactly 3")
	r.Equal(2, agentCount, "Agent count must remain exactly 2")
	r.Equal(2, entryCount, "Contest entry count must remain exactly 2")
}
