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
	TargetSchemaVersion = int64(1)
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

	// 3. Quick structural validation: ensure core tables exist
	requiredTables := []string{
		"roles", "users", "games", "contests", "agents",
		"submissions", "matches", "results", "rankings", "replays",
		"match_jobs", "match_runs", "contest_entries",
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

	return nil
}
