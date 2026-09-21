package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Role struct {
	Id          string `json:"id,omitempty" db:"id"`
	Description string `json:"description,omitempty" db:"description"`
	Active      bool   `json:"active,omitempty" db:"active"`
}

type Permission struct {
	Id          string `json:"id,omitempty" db:"id"`
	Description string `json:"description,omitempty" db:"description"`
	Active      bool   `json:"active,omitempty" db:"active"`
}

type User struct {
	Id           string    `json:"id,omitempty" db:"id"`
	Username     string    `json:"username,omitempty" db:"username"`
	Email        string    `json:"email,omitempty" db:"email"`
	Password     string    `json:"-" db:"password"` // Never expose password in json
	RoleId       string    `json:"role_id,omitempty" db:"role_id"`
	Roles        []string  `json:"roles,omitempty"`
	Active       bool      `json:"active,omitempty" db:"active"`
	CreatedAt    time.Time `json:"created_at,omitempty" db:"created_at"`
	Role         *Role     `json:"role,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
}

type Claims struct {
	UserID       string   `json:"user_id"`
	RoleId       string   `json:"role_id"`
	Roles        []string `json:"roles,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	jwt.RegisteredClaims
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token *Token `json:"token"`
	User  *User  `json:"user"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
