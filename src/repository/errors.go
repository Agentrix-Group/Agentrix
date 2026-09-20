package repository

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrFencingTokenMismatch  = errors.New("fencing token mismatch: job lease superseded or invalid")
	ErrMatchAlreadyCommitted = errors.New("match run already committed")
	ErrJobNotFound           = errors.New("match job not found")
	ErrMatchNotFound         = errors.New("match not found")

	ErrGameNotFound        = errors.New("game not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrParticipantNotFound = ErrUserNotFound
	ErrAgentAlreadyExists  = errors.New("agent already exists")
	ErrForeignKeyViolation = errors.New("foreign key violation")
	ErrSchemaIncompatible  = errors.New("database schema incompatible or missing tables")
)

type DBError struct {
	Err            error
	SQLState       string
	ConstraintName string
	SafeMessage    string
}

func (e *DBError) Error() string {
	if e.SafeMessage != "" {
		return fmt.Sprintf("%s (sql_state=%s, constraint=%s)", e.SafeMessage, e.SQLState, e.ConstraintName)
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "database error"
}

func (e *DBError) Unwrap() error {
	return e.Err
}

func ClassifyDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation
			if pgErr.ConstraintName == "agents_game_id_fkey" {
				return &DBError{
					Err:            ErrGameNotFound,
					SQLState:       pgErr.Code,
					ConstraintName: pgErr.ConstraintName,
					SafeMessage:    "game does not exist",
				}
			}
			if pgErr.ConstraintName == "agents_participant_id_fkey" || pgErr.ConstraintName == "agents_owner_user_id_fkey" {
				return &DBError{
					Err:            ErrUserNotFound,
					SQLState:       pgErr.Code,
					ConstraintName: pgErr.ConstraintName,
					SafeMessage:    "user does not exist",
				}
			}
			return &DBError{
				Err:            ErrForeignKeyViolation,
				SQLState:       pgErr.Code,
				ConstraintName: pgErr.ConstraintName,
				SafeMessage:    "referenced entity does not exist",
			}
		case "23505": // unique_violation
			return &DBError{
				Err:            ErrAgentAlreadyExists,
				SQLState:       pgErr.Code,
				ConstraintName: pgErr.ConstraintName,
				SafeMessage:    "entity already exists",
			}
		case "42P01", "42703": // undefined_table, undefined_column
			return &DBError{
				Err:            ErrSchemaIncompatible,
				SQLState:       pgErr.Code,
				ConstraintName: "",
				SafeMessage:    "database schema incompatible or unmigrated",
			}
		}
	}
	return err
}
