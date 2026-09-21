package service

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

// Session is the result of login or refresh. RefreshToken is delivered to
// the browser only as an HttpOnly cookie.
type Session struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
	User             *model.User
	Principal        *model.Principal
}

// dummyHash equalizes login timing for unknown usernames.
var dummyHash, _ = auth.HashPassword("agentrix-timing-equalizer")

// Principal resolves the actor of a request. An empty bearer yields the
// anonymous principal, whose capabilities are those of the spectator role.
func (s *Service) Principal(ctx context.Context, bearer string) (*model.Principal, error) {
	if bearer == "" {
		caps, err := s.store.CapabilitiesForRoles(ctx, []string{model.RoleSpectator})
		if err != nil {
			return nil, err
		}
		return &model.Principal{Capabilities: caps}, nil
	}
	claims, err := s.tokens.Parse(bearer, s.now())
	if err != nil {
		return nil, model.Unauthorized("invalid_token", "access token is invalid or expired")
	}
	return s.store.SessionPrincipal(ctx, claims.Subject, claims.SessionID, s.now())
}

func (s *Service) Register(ctx context.Context, username, email, password string) (*model.User, error) {
	if !s.opts.RegistrationOpen {
		return nil, model.Forbidden("registration_closed", "self registration is disabled")
	}
	username = strings.TrimSpace(username)
	email = model.NormalizeEmail(email)
	if err := errors.Join(model.ValidateUsername(username), model.ValidateEmail(email)); err != nil {
		return nil, firstDomainError(err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		if errors.Is(err, auth.ErrPasswordTooShort) || errors.Is(err, auth.ErrPasswordTooLong) {
			return nil, model.Validation("invalid_password", "%s", err.Error())
		}
		return nil, err
	}
	return s.createUser(ctx, "", username, email, hash, []string{model.RolePlayer})
}

// CreateUser is the administrative (and bootstrap) path to create accounts
// with explicit roles.
func (s *Service) CreateUser(ctx context.Context, actor model.Principal, username, email, password string, roles []string) (*model.User, error) {
	if err := requireCap(actor, model.CapUsersRolesManage); err != nil {
		return nil, err
	}
	username = strings.TrimSpace(username)
	email = model.NormalizeEmail(email)
	if err := errors.Join(model.ValidateUsername(username), model.ValidateEmail(email), validateRoles(roles)); err != nil {
		return nil, firstDomainError(err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, model.Validation("invalid_password", "%s", err.Error())
	}
	return s.createUser(ctx, actor.UserID, username, email, hash, roles)
}

// BootstrapAdmin creates the first administrator. It refuses to run when an
// admin already exists, so it cannot be used to escalate privileges.
func (s *Service) BootstrapAdmin(ctx context.Context, username, email, password string) (*model.User, error) {
	users, err := s.store.ListUsers(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if slices.Contains(u.Roles, model.RoleAdmin) {
			return nil, model.Conflict("admin_exists", "an administrator already exists")
		}
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, model.Validation("invalid_password", "%s", err.Error())
	}
	return s.createUser(ctx, "", strings.TrimSpace(username), model.NormalizeEmail(email), hash, []string{model.RoleAdmin})
}

func (s *Service) createUser(ctx context.Context, actorID, username, email, hash string, roles []string) (*model.User, error) {
	now := s.now()
	user := &model.User{ID: s.newID(), Username: username, Email: email, PasswordHash: hash, Status: model.UserActive,
		CreatedAt: now, UpdatedAt: now}
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		if err := q.CreateUser(ctx, user); err != nil {
			return err
		}
		if err := q.ReplaceUserRoles(ctx, user.ID, roles, actorID, now); err != nil {
			return err
		}
		return q.Audit(ctx, actorID, "user.created", "user", user.ID, map[string]any{"roles": roles}, now)
	})
	if err != nil {
		if domain, ok := model.AsError(err); ok && domain.Kind == model.KindConflict {
			return nil, model.Conflict("user_exists", "username or email is already registered")
		}
		return nil, err
	}
	user.Roles = append([]string(nil), roles...)
	sort.Strings(user.Roles)
	return user, nil
}

func (s *Service) Login(ctx context.Context, username, password, userAgent, ip string) (*Session, error) {
	invalid := model.Unauthorized("invalid_credentials", "invalid username or password")
	user, err := s.store.GetUserByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if model.KindOf(err) == model.KindNotFound {
			_, _, _ = auth.VerifyPassword(password, dummyHash)
			return nil, invalid
		}
		return nil, err
	}
	ok, needsRehash, err := auth.VerifyPassword(password, user.PasswordHash)
	if err != nil || !ok {
		return nil, invalid
	}
	if user.Status != model.UserActive {
		return nil, model.Forbidden("account_inactive", "account is %s", user.Status)
	}
	var session *Session
	err = s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		if needsRehash {
			hash, err := auth.HashPassword(password)
			if err != nil {
				return err
			}
			if err := q.UpdateUserPassword(ctx, user.ID, hash, now); err != nil {
				return err
			}
		}
		familyID := s.newID()
		family := repository.SessionFamily{ID: familyID, UserID: user.ID, CreatedAt: now,
			ExpiresAt: now.Add(s.opts.SessionMaxTTL), UserAgent: truncateText(userAgent, 255), IPAddress: truncateText(ip, 64)}
		if err := q.CreateSessionFamily(ctx, family); err != nil {
			return err
		}
		var err error
		session, _, err = s.issueSession(ctx, q, user, family, 1, now)
		if err != nil {
			return err
		}
		return q.PruneUserSessions(ctx, user.ID, s.opts.MaxSessions, now)
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) issueSession(ctx context.Context, q *repository.Queries, user *model.User, family repository.SessionFamily,
	generation int, now time.Time) (*Session, string, error) {
	raw, hash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, "", err
	}
	expires := now.Add(s.opts.RefreshIdleTTL)
	if family.ExpiresAt.Before(expires) {
		expires = family.ExpiresAt
	}
	token := repository.RefreshToken{ID: s.newID(), FamilyID: family.ID, Generation: generation, TokenHash: hash,
		CreatedAt: now, ExpiresAt: expires}
	if err := q.CreateRefreshToken(ctx, token); err != nil {
		return nil, "", err
	}
	access, accessExpires, err := s.tokens.Issue(user.ID, family.ID, now)
	if err != nil {
		return nil, "", err
	}
	caps, err := q.CapabilitiesForRoles(ctx, user.Roles)
	if err != nil {
		return nil, "", err
	}
	return &Session{AccessToken: access, AccessExpiresAt: accessExpires, RefreshToken: raw, RefreshExpiresAt: expires,
		User: user, Principal: &model.Principal{UserID: user.ID, SessionID: family.ID, Roles: user.Roles, Capabilities: caps},
	}, token.ID, nil
}

// Refresh rotates a refresh token exactly once. Both rows are locked, so two
// concurrent refreshes of the same token serialize: the first rotates, the
// second sees a used token, treats it as reuse and revokes the family.
func (s *Service) Refresh(ctx context.Context, rawToken string) (*Session, error) {
	invalid := model.Unauthorized("invalid_refresh_token", "refresh token is not valid")
	if !strings.HasPrefix(rawToken, auth.RefreshTokenPrefix) {
		return nil, invalid
	}
	var session *Session
	var outcome error
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		token, family, err := q.LockRefreshToken(ctx, auth.HashRefreshToken(rawToken))
		if err != nil {
			outcome = invalid
			return nil
		}
		switch {
		case family.RevokedAt != nil:
			outcome = invalid
			return nil
		case token.UsedAt != nil:
			// Reuse of a rotated token: revoke the whole family, commit that.
			outcome = model.Unauthorized("refresh_token_reused", "refresh token reuse detected; session revoked")
			if err := q.RevokeSessionFamily(ctx, family.ID, "reuse_detected", now); err != nil {
				return err
			}
			return q.Audit(ctx, family.UserID, "session.reuse_detected", "session", family.ID, nil, now)
		case !token.ExpiresAt.After(now) || !family.ExpiresAt.After(now):
			outcome = model.Unauthorized("session_expired", "session expired")
			return nil
		}
		user, err := q.GetUser(ctx, family.UserID)
		if err != nil {
			return err
		}
		if user.Status != model.UserActive {
			outcome = model.Forbidden("account_inactive", "account is %s", user.Status)
			return q.RevokeSessionFamily(ctx, family.ID, "user_disabled", now)
		}
		next, successorID, err := s.issueSession(ctx, q, user, *family, token.Generation+1, now)
		if err != nil {
			return err
		}
		if err := q.MarkRefreshTokenUsed(ctx, token.ID, successorID, now); err != nil {
			return err
		}
		session = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	if outcome != nil {
		return nil, outcome
	}
	return session, nil
}

// Logout revokes the session family identified by the refresh cookie, or by
// the access token when no cookie is available. It is idempotent.
func (s *Service) Logout(ctx context.Context, rawRefresh string, principal model.Principal) error {
	return s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		familyID := principal.SessionID
		if rawRefresh != "" {
			if _, family, err := q.LockRefreshToken(ctx, auth.HashRefreshToken(rawRefresh)); err == nil {
				if principal.UserID != "" && family.UserID != principal.UserID {
					return model.Forbidden("session_mismatch", "refresh token belongs to another account")
				}
				familyID = family.ID
			}
		}
		if familyID == "" {
			return nil
		}
		return q.RevokeSessionFamily(ctx, familyID, "logout", now)
	})
}

func (s *Service) LogoutAll(ctx context.Context, principal model.Principal) (int64, error) {
	if principal.Anonymous() {
		return 0, model.Unauthorized("authentication_required", "authentication is required")
	}
	var n int64
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		user, err := q.LockUser(ctx, principal.UserID)
		if err != nil {
			return err
		}
		if user.Status != model.UserActive {
			return model.Forbidden("account_inactive", "account is %s", user.Status)
		}
		n, err = q.RevokeUserSessions(ctx, principal.UserID, "logout_all", now)
		if err != nil {
			return err
		}
		return q.Audit(ctx, principal.UserID, "session.logout_all", "user", principal.UserID, map[string]any{"revoked": n}, now)
	})
	return n, err
}

func (s *Service) Me(ctx context.Context, p model.Principal) (*model.User, error) {
	if p.Anonymous() {
		return nil, model.Unauthorized("authentication_required", "authentication is required")
	}
	return s.store.GetUser(ctx, p.UserID)
}

func (s *Service) ListUsers(ctx context.Context, p model.Principal, status model.UserStatus) ([]model.User, error) {
	if err := requireCap(p, model.CapUsersReadAny); err != nil {
		return nil, err
	}
	if status != "" && !status.Valid() {
		return nil, model.Validation("invalid_status", "unknown user status %q", status)
	}
	return s.store.ListUsers(ctx, status)
}

func (s *Service) GetUser(ctx context.Context, p model.Principal, id string) (*model.User, error) {
	if !p.CanFor(id, model.CapUsersReadOwn, model.CapUsersReadAny) {
		return nil, model.NotFound("user_not_found", "user not found")
	}
	return s.store.GetUser(ctx, id)
}

// ReplaceUserRoles makes roles the exact role set of the user in one
// transaction. Capabilities are resolved per request, so the change is
// effective for the next request of every session of that user.
func (s *Service) ReplaceUserRoles(ctx context.Context, p model.Principal, userID string, roles []string) (*model.User, error) {
	if err := requireCap(p, model.CapUsersRolesManage); err != nil {
		return nil, err
	}
	roles = dedupe(roles)
	if err := validateRoles(roles); err != nil {
		return nil, err
	}
	if userID == p.UserID && !slices.Contains(roles, model.RoleAdmin) && slices.Contains(p.Roles, model.RoleAdmin) {
		return nil, model.Conflict("self_demotion", "administrators cannot remove their own admin role")
	}
	var updated *model.User
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		before, err := q.LockUser(ctx, userID)
		if err != nil {
			return err
		}
		if err := q.ReplaceUserRoles(ctx, userID, roles, p.UserID, now); err != nil {
			return err
		}
		if err := q.Audit(ctx, p.UserID, "user.roles_replaced", "user", userID,
			map[string]any{"before": before.Roles, "after": roles}, now); err != nil {
			return err
		}
		updated, err = q.GetUser(ctx, userID)
		return err
	})
	return updated, err
}

// SetUserStatus changes the account status. Leaving "active" revokes every
// session of the account in the same transaction.
func (s *Service) SetUserStatus(ctx context.Context, p model.Principal, userID string, status model.UserStatus) (*model.User, error) {
	if err := requireCap(p, model.CapUsersUpdateAny); err != nil {
		return nil, err
	}
	if !status.Valid() {
		return nil, model.Validation("invalid_status", "status must be active, suspended or disabled")
	}
	if userID == p.UserID {
		return nil, model.Conflict("self_status_change", "you cannot change the status of your own account")
	}
	var updated *model.User
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		before, err := q.LockUser(ctx, userID)
		if err != nil {
			return err
		}
		if before.Status == status {
			updated = before
			return nil
		}
		if err := q.UpdateUserStatus(ctx, userID, status, now); err != nil {
			return err
		}
		if status != model.UserActive {
			if _, err := q.RevokeUserSessions(ctx, userID, "user_disabled", now); err != nil {
				return err
			}
		}
		if err := q.Audit(ctx, p.UserID, "user.status_changed", "user", userID,
			map[string]any{"before": before.Status, "after": status}, now); err != nil {
			return err
		}
		updated, err = q.GetUser(ctx, userID)
		return err
	})
	return updated, err
}

// UpdateAccount changes email and/or password of the principal's account
// (or any account with users:update:any, except the password).
func (s *Service) UpdateAccount(ctx context.Context, p model.Principal, userID string, email *string, currentPassword, newPassword string) (*model.User, error) {
	if !p.CanFor(userID, model.CapUsersUpdateOwn, model.CapUsersUpdateAny) {
		return nil, model.NotFound("user_not_found", "user not found")
	}
	if newPassword != "" && userID != p.UserID {
		return nil, model.Forbidden("password_self_only", "only the account owner can change its password")
	}
	var updated *model.User
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		now := s.now()
		user, err := q.LockUser(ctx, userID)
		if err != nil {
			return err
		}
		if email != nil {
			normalized := model.NormalizeEmail(*email)
			if err := model.ValidateEmail(normalized); err != nil {
				return err
			}
			if err := q.UpdateUserEmail(ctx, userID, normalized, now); err != nil {
				return err
			}
		}
		if newPassword != "" {
			ok, _, err := auth.VerifyPassword(currentPassword, user.PasswordHash)
			if err != nil || !ok {
				return model.Forbidden("invalid_credentials", "current password is not correct")
			}
			hash, err := auth.HashPassword(newPassword)
			if err != nil {
				return model.Validation("invalid_password", "%s", err.Error())
			}
			if err := q.UpdateUserPassword(ctx, userID, hash, now); err != nil {
				return err
			}
		}
		if err := q.Audit(ctx, p.UserID, "user.account_updated", "user", userID,
			map[string]any{"email": email != nil, "password": newPassword != ""}, now); err != nil {
			return err
		}
		updated, err = q.GetUser(ctx, userID)
		return err
	})
	return updated, err
}

func validateRoles(roles []string) error {
	if len(roles) == 0 {
		return model.Validation("invalid_roles", "at least one role is required")
	}
	for _, r := range roles {
		if !model.ValidRole(r) {
			return model.Validation("invalid_roles", "unknown role %q", r)
		}
	}
	return nil
}

func dedupe(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return slices.Compact(out)
}

func firstDomainError(err error) error {
	if domain, ok := model.AsError(err); ok {
		return domain
	}
	return err
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "")
}
