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
	ErrSchemaDrift       = errors.New("database schema does not match the canonical baseline")
)

func init() {
	goose.SetBaseFS(MigrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		panic(fmt.Sprintf("failed to set goose dialect: %v", err))
	}
	goose.SetLogger(goose.NopLogger())
}

func Migrate(db *sql.DB) error { return goose.Up(db, MigrationsDir) }

func GetCurrentVersion(db *sql.DB) (int64, error) { return goose.GetDBVersion(db) }

func Status(db *sql.DB) error { return goose.Status(db, MigrationsDir) }

func Rollback(db *sql.DB) error { return goose.Down(db, MigrationsDir) }

// Reset rolls back every migration. It is only used by tests and explicit
// operator commands against disposable databases.
func Reset(db *sql.DB) error { return goose.Reset(db, MigrationsDir) }

// CriticalObjects lists constraints, unique indexes and triggers whose
// absence would let the database accept states the platform forbids.
var CriticalObjects = struct {
	Constraints []string
	Indexes     []string
	Triggers    []string
}{
	Constraints: []string{
		"uq_submissions_agent_version", "uq_submissions_identity",
		"uq_contest_entries_agent", "fk_contest_entries_contest", "fk_contest_entries_agent", "fk_contest_entries_submission",
		"ck_contests_window",
		"ck_matches_competitive_contest", "ck_matches_committed", "fk_matches_committed_run", "fk_matches_contest",
		"uq_match_slots_index", "fk_match_slots_match", "fk_match_slots_submission", "fk_match_slots_entry",
		"ck_match_slots_competitive_entry",
		"uq_match_runs_attempt", "ck_match_runs_reservation", "ck_match_runs_terminal",
		"uq_match_jobs_run", "fk_match_jobs_run", "ck_match_jobs_reserved",
		"uq_results_run_slot", "fk_results_run", "fk_results_slot",
		"uq_replays_run", "fk_replays_run", "ck_replays_published",
		"uq_ranking_snapshots_version", "uq_refresh_tokens_family_generation",
	},
	Indexes: []string{
		"uq_users_username_ci", "uq_users_email", "uq_refresh_tokens_hash",
		"uq_match_slots_competitive_submission", "uq_match_runs_committed", "uq_match_jobs_active",
	},
	Triggers: []string{
		"trg_submissions_immutable", "trg_contests_transition", "trg_contest_entries_roster",
		"trg_matches_transition", "trg_match_slots_immutable", "trg_match_runs_transition",
		"trg_match_jobs_transition", "trg_results_immutable", "trg_replays_transition",
		"trg_ranking_snapshots_immutable", "trg_audit_log_append_only",
	},
}

// CheckSchemaCompatible verifies the migration version and the presence of
// every critical constraint, index and trigger.
func CheckSchemaCompatible(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("database connection is nil")
	}
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database connectivity check failed: %w", err)
	}
	var hasVersionTable bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&hasVersionTable); err != nil {
		return fmt.Errorf("check migration table: %w", err)
	}
	if !hasVersionTable {
		return fmt.Errorf("%w: run the migrator (cmd/migrate up)", ErrSchemaMissing)
	}
	version, err := goose.GetDBVersion(db)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSchemaMissing, err)
	}
	if version < TargetSchemaVersion {
		return fmt.Errorf("%w: version %d, expected %d (run cmd/migrate up)", ErrPendingMigrations, version, TargetSchemaVersion)
	}
	if version > TargetSchemaVersion {
		return fmt.Errorf("%w: database version %d is newer than this binary (%d)", ErrSchemaDrift, version, TargetSchemaVersion)
	}
	checks := []struct {
		kind  string
		query string
		names []string
	}{
		{"constraint", `SELECT EXISTS (SELECT 1 FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace
			WHERE n.nspname = 'public' AND c.conname = $1 AND c.convalidated)`, CriticalObjects.Constraints},
		{"index", `SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = $1)`, CriticalObjects.Indexes},
		{"trigger", `SELECT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = $1 AND NOT tgisinternal AND tgenabled <> 'D')`, CriticalObjects.Triggers},
	}
	for _, check := range checks {
		for _, name := range check.names {
			var exists bool
			if err := db.QueryRowContext(ctx, check.query, name).Scan(&exists); err != nil {
				return fmt.Errorf("check %s %s: %w", check.kind, name, err)
			}
			if !exists {
				return fmt.Errorf("%w: %s %q is missing or disabled", ErrSchemaDrift, check.kind, name)
			}
		}
	}
	return nil
}
