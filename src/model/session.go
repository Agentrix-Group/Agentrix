package model

import "time"

// Session represents a persisted authentication session with refresh token family tracking.
type Session struct {
	Id         string    `json:"id" db:"id"`
	UserId     string    `json:"user_id" db:"user_id"`
	FamilyId   string    `json:"family_id" db:"family_id"`
	TokenHash  string    `json:"-" db:"token_hash"` // Cryptographic SHA-256 digest of the refresh secret
	UserAgent  string    `json:"user_agent" db:"user_agent"`
	IpAddress  string    `json:"ip_address" db:"ip_address"`
	IsRevoked  bool      `json:"is_revoked" db:"is_revoked"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ExpiresAt  time.Time `json:"expires_at" db:"expires_at"`
	LastUsedAt time.Time `json:"last_used_at" db:"last_used_at"`
}

// SessionClaims contains the context attributes attached to an authenticated session.
type SessionInfo struct {
	SessionId    string    `json:"session_id"`
	UserId       string    `json:"user_id"`
	Username     string    `json:"username"`
	Roles        []string  `json:"roles"`
	Capabilities []string  `json:"capabilities"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}
