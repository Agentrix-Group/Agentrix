// Package repository owns every SQL statement of Agentrix. It maps rows to
// model types and PostgreSQL errors to domain errors; business rules live in
// the service package. Multi-step use cases run inside Store.Tx.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbtx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Queries executes statements on either the pool or a transaction.
type Queries struct {
	db dbtx
}

type Store struct {
	DB *sql.DB
	*Queries
}

func NewStore(db *sql.DB) *Store {
	return &Store{DB: db, Queries: &Queries{db: db}}
}

// Tx runs fn in a READ COMMITTED transaction. Row locks (FOR UPDATE) inside
// fn serialize concurrent commands on the same aggregate.
func (s *Store) Tx(ctx context.Context, fn func(q *Queries) error) error {
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(&Queries{db: tx}); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback: %w", rbErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return classify(fmt.Errorf("commit transaction: %w", err))
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error { return s.DB.PingContext(ctx) }

// classify converts PostgreSQL constraint errors into domain errors with
// safe messages. Everything else stays an infrastructure error.
func classify(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return &model.Error{Kind: model.KindConflict, Code: "already_exists",
			Message: "a resource with the same identity already exists (" + pgErr.ConstraintName + ")", Err: err}
	case "23503":
		return &model.Error{Kind: model.KindValidation, Code: "reference_not_found",
			Message: "a referenced resource does not exist or does not match (" + pgErr.ConstraintName + ")", Err: err}
	case "23514":
		return &model.Error{Kind: model.KindConflict, Code: "invariant_violation",
			Message: "the change violates an integrity rule", Err: err}
	case "23502":
		return &model.Error{Kind: model.KindValidation, Code: "missing_field",
			Message: "a required field is missing (" + pgErr.ColumnName + ")", Err: err}
	case "22P02", "22001":
		return &model.Error{Kind: model.KindValidation, Code: "invalid_value", Message: "a value has an invalid format", Err: err}
	case "40001", "40P01":
		return &model.Error{Kind: model.KindConflict, Code: "concurrent_update", Message: "concurrent update, retry the request", Err: err}
	}
	return err
}

// expectOne turns an UPDATE/DELETE result into an error unless exactly one
// row was affected.
func expectOne(res sql.Result, err error, notFound error) error {
	if err != nil {
		return classify(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return notFound
	}
	return nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func strOrEmpty(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func timePtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}

func splitList(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}
