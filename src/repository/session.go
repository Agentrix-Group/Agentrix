package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
)

// SessionRepository defines the storage interface for persistent session management.
type SessionRepository interface {
	CreateSession(ctx context.Context, session *model.Session) error
	GetSessionByHash(ctx context.Context, tokenHash string) (*model.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	RevokeSessionFamily(ctx context.Context, familyID string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	PruneOldestUserSessions(ctx context.Context, userID string, keepCount int) error
	UpdateSessionLastUsed(ctx context.Context, sessionID string) error
}

func (r *repository) CreateSession(ctx context.Context, session *model.Session) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `
		INSERT INTO sessions (
			id, user_id, family_id, token_hash, user_agent, ip_address, 
			is_revoked, created_at, expires_at, last_used_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}
	if session.LastUsedAt.IsZero() {
		session.LastUsedAt = session.CreatedAt
	}

	_, err = db.ExecContext(ctx, query,
		session.Id,
		session.UserId,
		session.FamilyId,
		session.TokenHash,
		session.UserAgent,
		session.IpAddress,
		session.IsRevoked,
		session.CreatedAt,
		session.ExpiresAt,
		session.LastUsedAt,
	)
	if err != nil {
		return ClassifyDBError(err)
	}

	return nil
}

func (r *repository) GetSessionByHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	db, err := r.getDb()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, user_id, family_id, token_hash, user_agent, ip_address, 
		       is_revoked, created_at, expires_at, last_used_at
		FROM sessions
		WHERE token_hash = $1`

	var s model.Session
	err = db.QueryRowContext(ctx, query, tokenHash).Scan(
		&s.Id,
		&s.UserId,
		&s.FamilyId,
		&s.TokenHash,
		&s.UserAgent,
		&s.IpAddress,
		&s.IsRevoked,
		&s.CreatedAt,
		&s.ExpiresAt,
		&s.LastUsedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, ClassifyDBError(err)
	}

	return &s, nil
}

func (r *repository) RevokeSession(ctx context.Context, sessionID string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE sessions SET is_revoked = TRUE WHERE id = $1`
	_, err = db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return ClassifyDBError(err)
	}
	return nil
}

func (r *repository) RevokeSessionFamily(ctx context.Context, familyID string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE sessions SET is_revoked = TRUE WHERE family_id = $1`
	_, err = db.ExecContext(ctx, query, familyID)
	if err != nil {
		return ClassifyDBError(err)
	}
	return nil
}

func (r *repository) RevokeAllUserSessions(ctx context.Context, userID string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE sessions SET is_revoked = TRUE WHERE user_id = $1`
	_, err = db.ExecContext(ctx, query, userID)
	if err != nil {
		return ClassifyDBError(err)
	}
	return nil
}

func (r *repository) PruneOldestUserSessions(ctx context.Context, userID string, keepCount int) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	// Revoke active sessions beyond keepCount, ordered by created_at DESC
	query := `
		UPDATE sessions
		SET is_revoked = TRUE
		WHERE user_id = $1 AND is_revoked = FALSE AND id NOT IN (
			SELECT id FROM sessions
			WHERE user_id = $1 AND is_revoked = FALSE
			ORDER BY created_at DESC
			LIMIT $2
		)`

	_, err = db.ExecContext(ctx, query, userID, keepCount)
	if err != nil {
		return ClassifyDBError(err)
	}
	return nil
}

func (r *repository) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	db, err := r.getDb()
	if err != nil {
		return err
	}

	query := `UPDATE sessions SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err = db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return ClassifyDBError(err)
	}
	return nil
}
