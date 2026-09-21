package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/google/uuid"
)

var errUserNotFound = model.NotFound("user_not_found", "user not found")

const userColumns = `u.id, u.username, u.email, u.password_hash, u.status, u.created_at, u.updated_at,
	COALESCE((SELECT string_agg(ur.role_id, ',' ORDER BY ur.role_id) FROM user_roles ur WHERE ur.user_id = u.id), '')`

func scanUser(row interface{ Scan(...any) error }) (*model.User, error) {
	var u model.User
	var roles string
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Status, &u.CreatedAt, &u.UpdatedAt, &roles); err != nil {
		return nil, err
	}
	u.Roles = splitList(roles)
	return &u, nil
}

func (q *Queries) CreateUser(ctx context.Context, u *model.User) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO users (id, username, email, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)`,
		u.ID, u.Username, u.Email, u.PasswordHash, u.Status, u.CreatedAt)
	return classify(err)
}

func (q *Queries) GetUser(ctx context.Context, id string) (*model.User, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errUserNotFound
	}
	u, err := scanUser(q.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users u WHERE u.id = $1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errUserNotFound
	}
	return u, classify(err)
}

// LockUser locks the user row for the rest of the transaction.
func (q *Queries) LockUser(ctx context.Context, id string) (*model.User, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errUserNotFound
	}
	if _, err := q.db.ExecContext(ctx, `SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, id); err != nil {
		return nil, classify(err)
	}
	return q.GetUser(ctx, id)
}

func (q *Queries) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	u, err := scanUser(q.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users u WHERE lower(u.username) = lower($1)`, username))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errUserNotFound
	}
	return u, classify(err)
}

func (q *Queries) ListUsers(ctx context.Context, status model.UserStatus) ([]model.User, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT `+userColumns+` FROM users u
		WHERE ($1 = '' OR u.status = $1) ORDER BY u.created_at, u.id`, string(status))
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

func (q *Queries) UpdateUserStatus(ctx context.Context, id string, status model.UserStatus, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE users SET status = $2, updated_at = $3 WHERE id = $1`, id, status, now)
	return expectOne(res, err, errUserNotFound)
}

func (q *Queries) UpdateUserEmail(ctx context.Context, id, email string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE users SET email = $2, updated_at = $3 WHERE id = $1`, id, email, now)
	return expectOne(res, err, errUserNotFound)
}

func (q *Queries) UpdateUserPassword(ctx context.Context, id, hash string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1`, id, hash, now)
	return expectOne(res, err, errUserNotFound)
}

// ReplaceUserRoles makes roles the exact role set of the user. Must run in a
// transaction that locked the user row.
func (q *Queries) ReplaceUserRoles(ctx context.Context, userID string, roles []string, grantedBy string, now time.Time) error {
	if _, err := q.db.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id = $1 AND NOT (role_id = ANY($2::text[]))`,
		userID, roles); err != nil {
		return classify(err)
	}
	for _, role := range roles {
		if _, err := q.db.ExecContext(ctx, `
			INSERT INTO user_roles (user_id, role_id, granted_by, granted_at) VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id, role_id) DO NOTHING`, userID, role, nullString(grantedBy), now); err != nil {
			return classify(err)
		}
	}
	return nil
}

func (q *Queries) CapabilitiesForRoles(ctx context.Context, roles []string) ([]model.Capability, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT DISTINCT capability_id FROM role_capabilities
		WHERE role_id = ANY($1::text[]) ORDER BY capability_id`, roles)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []model.Capability
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, model.Capability(c))
	}
	return out, rows.Err()
}

func (q *Queries) ListCapabilityCatalog(ctx context.Context) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT id FROM capabilities ORDER BY id`)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SessionPrincipal resolves the principal of an access token: the session
// family must be live and the user active. Roles and capabilities are read
// from user_roles in the same statement, so revocations apply immediately.
func (q *Queries) SessionPrincipal(ctx context.Context, userID, familyID string, now time.Time) (*model.Principal, error) {
	var roles, caps string
	err := q.db.QueryRowContext(ctx, `
		SELECT
			COALESCE((SELECT string_agg(ur.role_id, ',' ORDER BY ur.role_id) FROM user_roles ur WHERE ur.user_id = u.id), ''),
			COALESCE((SELECT string_agg(DISTINCT rc.capability_id, ',' ORDER BY rc.capability_id)
			          FROM user_roles ur JOIN role_capabilities rc ON rc.role_id = ur.role_id
			          WHERE ur.user_id = u.id), '')
		FROM session_families f JOIN users u ON u.id = f.user_id
		WHERE f.id = $1 AND f.user_id = $2 AND f.revoked_at IS NULL AND f.expires_at > $3 AND u.status = 'active'`,
		familyID, userID, now).Scan(&roles, &caps)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.Unauthorized("session_revoked", "session is no longer valid")
	}
	if err != nil {
		return nil, classify(err)
	}
	p := &model.Principal{UserID: userID, SessionID: familyID, Roles: splitList(roles)}
	for _, c := range splitList(caps) {
		p.Capabilities = append(p.Capabilities, model.Capability(c))
	}
	return p, nil
}

type SessionFamily struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IPAddress string
}

type RefreshToken struct {
	ID         string
	FamilyID   string
	Generation int
	TokenHash  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	UsedAt     *time.Time
}

func (q *Queries) CreateSessionFamily(ctx context.Context, f SessionFamily) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO session_families (id, user_id, created_at, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)`, f.ID, f.UserID, f.CreatedAt, f.ExpiresAt, f.UserAgent, f.IPAddress)
	return classify(err)
}

func (q *Queries) CreateRefreshToken(ctx context.Context, t RefreshToken) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, family_id, generation, token_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, t.ID, t.FamilyID, t.Generation, t.TokenHash, t.CreatedAt, t.ExpiresAt)
	return classify(err)
}

// LockRefreshToken locks a refresh token and its family (FOR UPDATE) so two
// concurrent rotations of the same token serialize.
func (q *Queries) LockRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, *SessionFamily, error) {
	var t RefreshToken
	var f SessionFamily
	var used, revoked sql.NullTime
	err := q.db.QueryRowContext(ctx, `
		SELECT t.id, t.family_id, t.generation, t.token_hash, t.created_at, t.expires_at, t.used_at,
		       f.id, f.user_id, f.created_at, f.expires_at, f.revoked_at, f.user_agent, f.ip_address
		FROM refresh_tokens t JOIN session_families f ON f.id = t.family_id
		WHERE t.token_hash = $1
		FOR UPDATE OF t, f`, tokenHash).Scan(
		&t.ID, &t.FamilyID, &t.Generation, &t.TokenHash, &t.CreatedAt, &t.ExpiresAt, &used,
		&f.ID, &f.UserID, &f.CreatedAt, &f.ExpiresAt, &revoked, &f.UserAgent, &f.IPAddress)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, model.Unauthorized("invalid_refresh_token", "refresh token is not valid")
	}
	if err != nil {
		return nil, nil, classify(err)
	}
	t.UsedAt = timePtr(used)
	f.RevokedAt = timePtr(revoked)
	return &t, &f, nil
}

func (q *Queries) MarkRefreshTokenUsed(ctx context.Context, id, replacedBy string, now time.Time) error {
	res, err := q.db.ExecContext(ctx, `UPDATE refresh_tokens SET used_at = $2, replaced_by = $3
		WHERE id = $1 AND used_at IS NULL`, id, now, nullString(replacedBy))
	return expectOne(res, err, model.Unauthorized("invalid_refresh_token", "refresh token was already used"))
}

func (q *Queries) RevokeSessionFamily(ctx context.Context, familyID, reason string, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `UPDATE session_families SET revoked_at = $2, revoke_reason = $3
		WHERE id = $1 AND revoked_at IS NULL`, familyID, now, reason)
	return classify(err)
}

func (q *Queries) RevokeUserSessions(ctx context.Context, userID, reason string, now time.Time) (int64, error) {
	res, err := q.db.ExecContext(ctx, `UPDATE session_families SET revoked_at = $2, revoke_reason = $3
		WHERE user_id = $1 AND revoked_at IS NULL`, userID, now, reason)
	if err != nil {
		return 0, classify(err)
	}
	return res.RowsAffected()
}

// PruneUserSessions revokes the oldest live families beyond keep.
func (q *Queries) PruneUserSessions(ctx context.Context, userID string, keep int, now time.Time) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE session_families SET revoked_at = $3, revoke_reason = 'pruned'
		WHERE id IN (
			SELECT id FROM session_families
			WHERE user_id = $1 AND revoked_at IS NULL
			ORDER BY created_at DESC, id DESC OFFSET $2)`, userID, keep, now)
	return classify(err)
}

func (q *Queries) Audit(ctx context.Context, actorID, action, resourceType, resourceID string, details map[string]any, now time.Time) error {
	if details == nil {
		details = map[string]any{}
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = q.db.ExecContext(ctx, `
		INSERT INTO audit_log (id, actor_user_id, action, resource_type, resource_id, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.NewString(), nullString(actorID), action, resourceType, resourceID, raw, now)
	return classify(err)
}

type AuditEntry struct {
	ActorUserID  string
	Action       string
	ResourceType string
	ResourceID   string
	Details      json.RawMessage
	CreatedAt    time.Time
}

func (q *Queries) ListAudit(ctx context.Context, resourceType, resourceID string) ([]AuditEntry, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT COALESCE(actor_user_id::text, ''), action, resource_type, resource_id, details, created_at
		FROM audit_log WHERE resource_type = $1 AND resource_id = $2 ORDER BY created_at, id`, resourceType, resourceID)
	if err != nil {
		return nil, classify(err)
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ActorUserID, &e.Action, &e.ResourceType, &e.ResourceID, &e.Details, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
