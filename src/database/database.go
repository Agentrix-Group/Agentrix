package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var MigrationsFS embed.FS

const (
	MigrationsDir       = "migrations"
	TargetSchemaVersion = int64(4)
)

var (
	ErrPendingMigrations = errors.New("database schema has pending migrations")
	ErrSchemaMissing     = errors.New("database schema is not initialized")
)

func init() {
	goose.SetBaseFS(MigrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		panic(fmt.Sprintf("failed to set goose dialect: %v", err))
	}
}

// Migrate applies all pending migrations forward.
func Migrate(db *sql.DB) error {
	return goose.Up(db, MigrationsDir)
}

// GetCurrentVersion returns the latest applied migration version in the database.
func GetCurrentVersion(db *sql.DB) (int64, error) {
	return goose.GetDBVersion(db)
}

// Status prints the migration status to standard output.
func Status(db *sql.DB) error {
	return goose.Status(db, MigrationsDir)
}

// Rollback rolls back the most recently applied migration.
func Rollback(db *sql.DB) error {
	return goose.Down(db, MigrationsDir)
}

// Reset rolls back all applied migrations.
func Reset(db *sql.DB) error {
	return goose.Reset(db, MigrationsDir)
}

// CheckSchemaCompatible verifies that the database has all required migrations applied.
func CheckSchemaCompatible(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("database connection is nil")
	}

	// 1. Verify basic connectivity
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connectivity check failed: %w", err)
	}

	// 2. Check goose version table
	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("%w: goose_db_version unreadable: %v (run 'make db-migrate')", ErrSchemaMissing, err)
	}

	if currentVersion < TargetSchemaVersion {
		return fmt.Errorf("%w: current version is %d, expected version %d (run 'make db-migrate')",
			ErrPendingMigrations, currentVersion, TargetSchemaVersion)
	}

	// 3. Structural validation: ensure core tables exist
	requiredTables := []string{
		"roles", "permissions", "role_permissions", "user_roles",
		"users", "sessions", "games", "contests", "agents", "submissions",
		"contest_entries", "rankings", "matches", "match_slots",
		"results", "replays", "match_jobs", "match_runs",
		"ranking_applied_runs", "contest_rankings_snapshots",
	}

	for _, table := range requiredTables {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		)`
		if err := db.QueryRowContext(ctx, query, table).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("%w: required table %q is missing (run 'make db-migrate')", ErrSchemaMissing, table)
		}
	}

	// 4. Column validation: ensure critical schema columns exist
	requiredColumns := []struct {
		Table  string
		Column string
	}{
		{"contest_entries", "submission_id"},
		{"match_slots", "submission_id"},
		{"match_slots", "slot_index"},
		{"results", "match_run_id"},
		{"results", "slot_id"},
		{"replays", "match_run_id"},
		{"matches", "committed_run_id"},
		{"contests", "scoring_policy"},
		{"rankings", "points"},
		{"rankings", "disqualifications"},
		{"rankings", "tiebreaker_score"},
	}

	for _, col := range requiredColumns {
		var exists bool
		query := `SELECT EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
		)`
		if err := db.QueryRowContext(ctx, query, col.Table, col.Column).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check column %s.%s: %w", col.Table, col.Column, err)
		}
		if !exists {
			return fmt.Errorf("%w: required column %q.%q is missing (run 'make db-migrate')", ErrSchemaMissing, col.Table, col.Column)
		}
	}

	return nil
}
