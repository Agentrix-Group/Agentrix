package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	Issuer   = "agentrix-api"
	Audience = "agentrix"
	// RefreshTokenPrefix makes leaked refresh tokens easy to recognize.
	RefreshTokenPrefix = "artx_rf_"
)

var ErrInvalidToken = errors.New("invalid access token")

// AccessClaims is the complete content of an access token. sid is the
// session family; the API rejects the token as soon as that family is
// revoked, so logout and role changes take effect before expiry.
type AccessClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
	ttl    time.Duration
}

func NewTokens(secret []byte, ttl time.Duration) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("access token secret must contain at least 32 bytes")
	}
	if ttl <= 0 {
		return nil, errors.New("access token ttl must be positive")
	}
	return &Tokens{secret: append([]byte(nil), secret...), ttl: ttl}, nil
}

func (t *Tokens) TTL() time.Duration { return t.ttl }

func (t *Tokens) Issue(userID, sessionID string, now time.Time) (string, time.Time, error) {
	expires := now.Add(t.ttl)
	claims := AccessClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expires, nil
}

// Parse validates signature (HS256 only), issuer, audience and time claims.
func (t *Tokens) Parse(token string, now time.Time) (*AccessClaims, error) {
	claims := &AccessClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil || !parsed.Valid || claims.Subject == "" || claims.SessionID == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// NewRefreshToken returns an opaque random refresh token and its SHA-256.
func NewRefreshToken() (token, hash string, err error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}
	token = RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(raw)
	return token, HashRefreshToken(token), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
