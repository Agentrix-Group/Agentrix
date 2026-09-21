package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/google/uuid"
)

var (
	ErrSessionNotFound      = errors.New("session not found or invalid")
	ErrSessionExpired       = errors.New("session has expired")
	ErrSessionRevoked       = errors.New("session has been revoked")
	ErrSessionReuseDetected = errors.New("refresh token reuse detected: session family revoked for security")
	ErrAccountDisabled      = errors.New("user account is inactive or disabled")
)

const (
	DefaultMaxSessionsPerUser = 10
	RefreshTokenPrefix        = "artx_rf_"
)

// SessionManager coordinates stateful refresh token persistence, rotation,
// reuse detection, and family invalidation.
type SessionManager struct {
	repo        repository.Repository
	auth        *Auth
	maxSessions int
}

func NewSessionManager(repo repository.Repository, auth *Auth) *SessionManager {
	return &SessionManager{
		repo:        repo,
		auth:        auth,
		maxSessions: DefaultMaxSessionsPerUser,
	}
}

// CreateSession generates a new session and refresh token in a new token family.
func (sm *SessionManager) CreateSession(ctx context.Context, user *model.User, userAgent, ipAddress string) (*model.Token, error) {
	familyID := uuid.New().String()
	return sm.issueSessionToken(ctx, user, familyID, userAgent, ipAddress)
}

// RefreshSession performs atomic one-time rotation of a refresh token.
// If an already revoked token is used, it detects reuse and immediately revokes
// the entire token family to prevent session hijacking.
func (sm *SessionManager) RefreshSession(ctx context.Context, rawRefreshToken, userAgent, ipAddress string) (*model.Token, error) {
	if rawRefreshToken == "" {
		return nil, ErrSessionNotFound
	}

	tokenHash := HashRefreshToken(rawRefreshToken)
	session, err := sm.repo.GetSessionByHash(ctx, tokenHash)
	if err != nil {
		// Fallback check: could this be a valid legacy stateless JWT refresh token?
		claims, jwtErr := sm.auth.ValidateRefreshToken(rawRefreshToken)
		if jwtErr == nil && claims != nil {
			// Legacy token detected -> upgrade user to stateful session
			user, uErr := sm.repo.GetUser(ctx, claims.UserID)
			if uErr != nil || !user.Active {
				return nil, ErrAccountDisabled
			}
			return sm.CreateSession(ctx, user, userAgent, ipAddress)
		}
		return nil, ErrSessionNotFound
	}

	// 1. Reuse detection: if this session was already revoked, someone is replaying a used token!
	if session.IsRevoked {
		// Revoke the entire family immediately to protect the compromised user
		_ = sm.repo.RevokeSessionFamily(ctx, session.FamilyId)
		return nil, ErrSessionReuseDetected
	}

	// 2. Check expiration
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = sm.repo.RevokeSession(ctx, session.Id)
		return nil, ErrSessionExpired
	}

	// 3. Verify user is still active in database
	user, err := sm.repo.GetUser(ctx, session.UserId)
	if err != nil || !user.Active {
		_ = sm.repo.RevokeSessionFamily(ctx, session.FamilyId)
		return nil, ErrAccountDisabled
	}

	// 4. Invalidate the current refresh token (one-time use)
	if err := sm.repo.RevokeSession(ctx, session.Id); err != nil {
		return nil, fmt.Errorf("failed to revoke rotated session: %w", err)
	}

	// 5. Issue next token in the SAME family
	return sm.issueSessionToken(ctx, user, session.FamilyId, userAgent, ipAddress)
}

// RevokeSession revokes a single active session (e.g. standard logout).
func (sm *SessionManager) RevokeSession(ctx context.Context, rawRefreshToken string) error {
	tokenHash := HashRefreshToken(rawRefreshToken)
	session, err := sm.repo.GetSessionByHash(ctx, tokenHash)
	if err != nil {
		return nil // Session already nonexistent or invalid
	}
	return sm.repo.RevokeSessionFamily(ctx, session.FamilyId)
}

// RevokeAllUserSessions revokes all sessions for a user (e.g. global logout).
func (sm *SessionManager) RevokeAllUserSessions(ctx context.Context, userID string) error {
	return sm.repo.RevokeAllUserSessions(ctx, userID)
}

// issueSessionToken creates a persisted session record and generates the token pair.
func (sm *SessionManager) issueSessionToken(
	ctx context.Context,
	user *model.User,
	familyID string,
	userAgent, ipAddress string,
) (*model.Token, error) {
	// Generate 32 bytes of cryptographic randomness for the refresh secret
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random session secret: %w", err)
	}

	rawRefreshToken := RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(rawBytes)
	tokenHash := HashRefreshToken(rawRefreshToken)

	now := time.Now().UTC()
	sessionExpiry := now.Add(RefreshTokenDuration)

	session := &model.Session{
		Id:         uuid.New().String(),
		UserId:     user.Id,
		FamilyId:   familyID,
		TokenHash:  tokenHash,
		UserAgent:  userAgent,
		IpAddress:  ipAddress,
		IsRevoked:  false,
		CreatedAt:  now,
		ExpiresAt:  sessionExpiry,
		LastUsedAt: now,
	}

	if err := sm.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to persist session: %w", err)
	}

	// Prune old sessions beyond the allowed limit
	_ = sm.repo.PruneOldestUserSessions(ctx, user.Id, sm.maxSessions)

	// Generate access token carrying user ID, roles, and effective capabilities
	roles := user.Roles
	if len(roles) == 0 && user.RoleId != "" {
		roles = []string{user.RoleId}
	}
	accessToken, err := sm.auth.GenerateAccessToken(user.Id, user.RoleId, roles, user.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	return &model.Token{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		ExpiresIn:    int64(AccessTokenDuration.Seconds()),
		TokenType:    TokenType,
	}, nil
}

// HashRefreshToken calculates the SHA-256 cryptographic digest of a refresh token secret.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", sum)
}
