package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestUserAndAuthModels(t *testing.T) {
	r := require.New(t)

	now := time.Now().UTC().Truncate(time.Second)
	u := User{
		Id:        "user-1",
		Username:  "jdaniel",
		Email:     "jd@example.com",
		Password:  "hashed",
		RoleId:    "admin",
		Active:    true,
		CreatedAt: now,
		Role: &Role{
			Id:          "admin",
			Description: "Administrator",
			Active:      true,
		},
	}

	data, err := json.Marshal(u)
	r.NoError(err)
	r.Contains(string(data), `"username":"jdaniel"`)
	r.Contains(string(data), `"email":"jd@example.com"`)

	var parsed User
	err = json.Unmarshal(data, &parsed)
	r.NoError(err)
	r.Equal("user-1", parsed.Id)
	r.Equal("jdaniel", parsed.Username)
	r.NotNil(parsed.Role)
	r.Equal("Administrator", parsed.Role.Description)

	// Claims test
	claims := Claims{
		UserID: "user-1",
		RoleId: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	claimsData, err := json.Marshal(claims)
	r.NoError(err)
	r.Contains(string(claimsData), `"user_id":"user-1"`)
	r.Contains(string(claimsData), `"role_id":"admin"`)

	// Login & Register requests
	loginReq := LoginRequest{Username: "jdaniel", Password: "secret"}
	loginData, err := json.Marshal(loginReq)
	r.NoError(err)
	r.Contains(string(loginData), `"username":"jdaniel"`)

	regReq := RegisterRequest{Username: "jdaniel", Email: "jd@example.com", Password: "secret"}
	regData, err := json.Marshal(regReq)
	r.NoError(err)
	r.Contains(string(regData), `"email":"jd@example.com"`)
}
