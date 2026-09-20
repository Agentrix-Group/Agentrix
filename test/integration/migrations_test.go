package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

// Tests: Fresh database migration from scratch
func TestIntegration_Migrations_FreshDatabase(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_fresh_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Initially, schema is not compatible
	err := database.CheckSchemaCompatible(ctx, conn.Db)
	r.Error(err, "Fresh empty DB must not declare schema compatible")

	// 2. Apply migrations forward
	err = database.Migrate(conn.Db)
	r.NoError(err, "Migrate must succeed on empty database")

	// 3. Current version must match TargetSchemaVersion
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(database.TargetSchemaVersion, ver, "Schema version must equal TargetSchemaVersion (1)")

	// 4. Schema is now compatible
	err = database.CheckSchemaCompatible(ctx, conn.Db)
	r.NoError(err, "CheckSchemaCompatible must pass after migration")

	// 5. Verify core tables exist
	coreTables := []string{
		"roles", "permissions", "role_permissions", "users", "categories",
		"games", "contests", "agents", "submissions", "matches",
		"results", "rankings", "replays", "match_jobs", "match_runs",
	}
	for _, table := range coreTables {
		var exists bool
		err := conn.Db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' AND table_name = $1
			)`, table).Scan(&exists)
		r.NoError(err)
		r.True(exists, "Table %s must exist after migration", table)
	}

	// 6. Verify replay columns exist
	var hasSha256, hasSizeBytes bool
	err = conn.Db.QueryRow(`
		SELECT 
			EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'replays' AND column_name = 'sha256'),
			EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'replays' AND column_name = 'size_bytes')
	`).Scan(&hasSha256, &hasSizeBytes)
	r.NoError(err)
	r.True(hasSha256, "replays.sha256 must exist")
	r.True(hasSizeBytes, "replays.size_bytes must exist")

	// 7. Verify match_jobs columns exist
	var hasRunId, hasFencingToken, hasLeaseUntil bool
	err = conn.Db.QueryRow(`
		SELECT 
			EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'match_jobs' AND column_name = 'run_id'),
			EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'match_jobs' AND column_name = 'fencing_token'),
			EXISTS (SELECT FROM information_schema.columns WHERE table_name = 'match_jobs' AND column_name = 'lease_until')
	`).Scan(&hasRunId, &hasFencingToken, &hasLeaseUntil)
	r.NoError(err)
	r.True(hasRunId, "match_jobs.run_id must exist")
	r.True(hasFencingToken, "match_jobs.fencing_token must exist")
	r.True(hasLeaseUntil, "match_jobs.lease_until must exist")

	// 8. Idempotency test: Applying migrations a second time does not alter data or error
	err = database.Migrate(conn.Db)
	r.NoError(err, "Second migration run must succeed idempotently")

	ver2, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(ver, ver2, "Version must remain identical after second migration run")
}

// Tests: Canonical schema migration and seed integrity
func TestIntegration_Migrations_CanonicalDatabaseBootstrapAndIntegrity(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_bootstrap_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Initially, schema is not compatible
	err := database.CheckSchemaCompatible(ctx, conn.Db)
	r.Error(err, "Unmigrated empty database must fail schema compatibility check")

	// 2. Apply canonical Goose migration
	err = database.Migrate(conn.Db)
	r.NoError(err, "Migrating canonical schema must succeed without errors")

	// 3. Verify logical version reached TargetSchemaVersion (1)
	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(database.TargetSchemaVersion, ver, "Database version must match TargetSchemaVersion (1)")

	// 4. Verify all canonical tables exist
	canonicalTables := []string{
		"roles", "permissions", "role_permissions", "users", "categories",
		"games", "contests", "agents", "submissions", "matches",
		"results", "rankings", "replays", "match_jobs", "match_runs",
		"contest_entries",
	}
	for _, table := range canonicalTables {
		var exists bool
		err := conn.Db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' AND table_name = $1
			)`, table).Scan(&exists)
		r.NoError(err)
		r.True(exists, "Table %s must exist in canonical schema", table)
	}

	// 5. Apply canonical seeds
	seedSQL, err := os.ReadFile("../../script/data/00_seeds_postgresql.sql")
	r.NoError(err)
	_, err = conn.Db.Exec(string(seedSQL))
	r.NoError(err, "Applying canonical seeds must succeed")

	// 6. Schema check passes now
	err = database.CheckSchemaCompatible(ctx, conn.Db)
	r.NoError(err, "Schema must be compatible after migration")

	// 7. Idempotency test: Re-running migration and seeds succeeds with zero errors
	err = database.Migrate(conn.Db)
	r.NoError(err, "Re-migrating canonical database must succeed idempotently")

	ver2, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(ver, ver2, "Schema version must remain stable")

	_, err = conn.Db.Exec(string(seedSQL))
	r.NoError(err, "Re-executing seeds must succeed idempotently")
}

// Tests: Health probes /health/live and /health/ready
func TestIntegration_Migrations_HealthAndReadinessProbes(t *testing.T) {
	r := require.New(t)

	dbName := fmt.Sprintf("agentrix_health_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Initial state: database empty (no migrations, no seeds)
	srv, _ := setupTestServer(conn)

	// Liveness must pass even if DB is not ready
	{
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var liveResp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &liveResp)
		r.Equal("UP", liveResp["status"])
	}

	// Readiness must FAIL (503 NOT_READY) because migrations and starfighter are missing
	{
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusServiceUnavailable, rec.Code)

		var readyResp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &readyResp)
		r.Equal("NOT_READY", readyResp["status"])
	}

	// 2. Apply migrations
	err := database.Migrate(conn.Db)
	r.NoError(err)

	// Readiness must still FAIL because starfighter game row is not yet seeded
	{
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusServiceUnavailable, rec.Code, "Readiness must fail without active starfighter game")
	}

	// 3. Seed starfighter game
	_, err = conn.Db.Exec(`
		INSERT INTO games (id, name, description, active)
		VALUES ('starfighter', 'Starfighter Arena', 'Active game', TRUE);
	`)
	r.NoError(err)

	// Setup artifact store for server
	artifactsDir, err := os.MkdirTemp("", "agentrix_test_artifacts_*")
	r.NoError(err)
	defer os.RemoveAll(artifactsDir)

	cfg := getTestPostgresConfig(dbName)
	cfg.Artifacts.Dir = artifactsDir
	artStore, err := connection.NewArtifactStore(context.Background(), cfg)
	r.NoError(err)

	// Recreate server with artifact store
	srv, _ = setupTestServer(conn)
	srv.Service = service.NewService(repository.NewRepository(conn), artStore, connection.NewJobQueue(10), nil)

	// Readiness must now PASS (200 READY)
	{
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		r.Equal(http.StatusOK, rec.Code)

		var readyResp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &readyResp)
		r.Equal("READY", readyResp["status"])
		checks, ok := readyResp["checks"].(map[string]any)
		r.True(ok)
		r.Equal("UP", checks["database"])
		r.Equal("UP", checks["schema"])
		r.Equal("UP", checks["artifacts"])
		r.Equal("UP", checks["starfighter"])
	}
}
