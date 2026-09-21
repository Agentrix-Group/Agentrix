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
		"contest_entries", "sessions",
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

// Tests: Legacy participant schema migration, user_roles backfill, and historical protection
func TestIntegration_Migrations_LegacyParticipantToUserMigration(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_legacy_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Simulate a legacy database with legacy participants table before Goose runs
	_, err := conn.Db.Exec(`
		CREATE TABLE participants (
			id VARCHAR(64) PRIMARY KEY,
			username VARCHAR(64) NOT NULL UNIQUE,
			email VARCHAR(128) NOT NULL UNIQUE,
			password VARCHAR(255) NOT NULL,
			role_id VARCHAR(64) DEFAULT 'participant',
			active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);

		INSERT INTO participants (id, username, email, password, role_id, active) VALUES
		('part-legacy-01', 'legacy_pilot', 'legacy@agentrix.local', 'legacy_hash', 'participant', TRUE);
	`)
	r.NoError(err, "Legacy participants table must be created")

	// 2. Run migrations forward to canonical version 2
	err = database.Migrate(conn.Db)
	r.NoError(err, "Migrating legacy database forward must succeed")

	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(database.TargetSchemaVersion, ver)

	// 3. Verify user migrated from participants
	var uUsername, uEmail, uRoleId string
	var uActive bool
	err = conn.Db.QueryRow(`
		SELECT username, email, role_id, active FROM users WHERE id = 'part-legacy-01'
	`).Scan(&uUsername, &uEmail, &uRoleId, &uActive)
	r.NoError(err, "Migrated user must exist in users table")
	r.Equal("legacy_pilot", uUsername)
	r.Equal("legacy@agentrix.local", uEmail)
	r.True(uActive)

	// 4. Verify user_roles populated
	var hasRole bool
	err = conn.Db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM user_roles WHERE user_id = 'part-legacy-01' AND role_id = 'participant'
		)
	`).Scan(&hasRole)
	r.NoError(err)
	r.True(hasRole, "user_roles must be backfilled from users.role_id")

	// 5. Verify match_slots table and slots functionality
	_, err = conn.Db.Exec(`
		INSERT INTO games (id, name, description) VALUES ('starfighter', 'Starfighter Arena', 'Game') ON CONFLICT DO NOTHING;
		INSERT INTO matches (id, game_id, status) VALUES ('match-slot-test', 'starfighter', 'scheduled');
		INSERT INTO agents (id, owner_user_id, game_id, name) VALUES ('agent-slot-test', 'part-legacy-01', 'starfighter', 'SlotBot');
		INSERT INTO submissions (id, agent_id, version, status) VALUES ('sub-slot-test', 'agent-slot-test', 1, 'ready');
		INSERT INTO match_slots (id, match_id, slot_index, submission_id, agent_name, username) VALUES
		('slot-1', 'match-slot-test', 0, 'sub-slot-test', 'SlotBot', 'legacy_pilot');
	`)
	r.NoError(err, "Match slots must be functional")

	// Duplicate slot_index in same match must fail unique constraint uk_match_slots_index
	_, err = conn.Db.Exec(`
		INSERT INTO match_slots (id, match_id, slot_index, submission_id, agent_name, username) VALUES
		('slot-dup', 'match-slot-test', 0, 'sub-slot-test', 'SlotBot', 'legacy_pilot');
	`)
	r.Error(err, "Duplicate slot_index for same match must violate uk_match_slots_index")

	// 6. Historical Data Protection: Deleting game when matches exist must be restricted
	_, err = conn.Db.Exec(`DELETE FROM games WHERE id = 'starfighter'`)
	r.Error(err, "Deleting game with active matches must be blocked by ON DELETE RESTRICT foreign key")

	_ = ctx
}

// Tests: Full reversibility (Down) and idempotence across all migration versions
func TestIntegration_Migrations_ReversibilityAndIdempotence(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	dbName := fmt.Sprintf("agentrix_rev_%d", time.Now().UnixNano()%1000000)
	conn, cleanup := createIsolatedDB(t, dbName)
	defer cleanup()

	// 1. Initial Migrate: 0 -> 4
	err := database.Migrate(conn.Db)
	r.NoError(err, "Initial forward migration to v4 must succeed")

	ver, err := database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(int64(4), ver)

	// 2. Idempotence: re-running Migrate on up-to-date DB must succeed
	err = database.Migrate(conn.Db)
	r.NoError(err, "Re-running Migrate on current schema must be idempotent")

	ver, err = database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(int64(4), ver)

	// 3. Rollback v4 -> v3
	err = database.Rollback(conn.Db)
	r.NoError(err, "Rollback from v4 to v3 must succeed")

	ver, err = database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(int64(3), ver)

	var snapTableExists bool
	_ = conn.Db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'contest_rankings_snapshots'
		)`).Scan(&snapTableExists)
	r.False(snapTableExists, "contest_rankings_snapshots must be dropped after rolling back v4")

	// 4. Rollback v3 -> v2
	err = database.Rollback(conn.Db)
	r.NoError(err, "Rollback from v3 to v2 must succeed")

	ver, err = database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(int64(2), ver)

	var sessionsTableExists bool
	_ = conn.Db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'sessions'
		)`).Scan(&sessionsTableExists)
	r.False(sessionsTableExists, "sessions table must be dropped after rolling back v3")

	// 5. Re-apply all migrations forward: v2 -> v4
	err = database.Migrate(conn.Db)
	r.NoError(err, "Re-migrating from v2 back to v4 must succeed cleanly")

	ver, err = database.GetCurrentVersion(conn.Db)
	r.NoError(err)
	r.Equal(int64(4), ver)

	err = database.CheckSchemaCompatible(ctx, conn.Db)
	r.NoError(err, "Database must be fully schema compatible after re-migrating")
}
