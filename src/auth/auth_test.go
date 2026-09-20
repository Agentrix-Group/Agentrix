package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	r := require.New(t)
	auth := NewAuth("test-access-secret-key", "test-refresh-secret-key")

	testCases := []struct {
		name   string
		userId string
		roleId string
	}{
		{name: "Valid inputs", userId: "user123", roleId: "admin"},
		{name: "Player inputs", userId: "user-456", roleId: "player"},
		{name: "Special characters", userId: "user@domain.com", roleId: "referee"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := auth.GenerateAuthToken(tc.userId, tc.roleId)
			r.NoError(err)
			r.NotNil(token)
			r.NotEmpty(token.AccessToken)
			r.NotEmpty(token.RefreshToken)
			r.Equal(TokenType, token.TokenType)
			r.Equal(int64(AccessTokenDuration.Seconds()), token.ExpiresIn)
		})
	}
}

func TestValidateToken(t *testing.T) {
	r := require.New(t)
	auth := NewAuth("test-access-secret-key", "test-refresh-secret-key")

	expectedUserId := "user123"
	expectedRoleId := "admin"

	validToken, err := auth.GenerateAuthToken(expectedUserId, expectedRoleId)
	r.NoError(err)

	testCases := []struct {
		name         string
		accessToken  string
		refreshToken string
		wantErr      bool
	}{
		{name: "Valid tokens", accessToken: validToken.AccessToken, refreshToken: validToken.RefreshToken, wantErr: false},
		{name: "Empty tokens", accessToken: "", refreshToken: "", wantErr: true},
		{name: "Invalid tokens", accessToken: "invalid.token.here", refreshToken: "invalid.token.here", wantErr: true},
		{name: "Malformed tokens", accessToken: "not-a-jwt-token", refreshToken: "not-a-jwt-token", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			accessClaims, accessErr := auth.ValidateAccessToken(tc.accessToken)
			if tc.wantErr {
				r.Error(accessErr)
			} else {
				r.NoError(accessErr)
				r.Equal(expectedUserId, accessClaims.UserID)
				r.Equal(expectedRoleId, accessClaims.RoleId)
			}

			refreshClaims, refreshErr := auth.ValidateRefreshToken(tc.refreshToken)
			if tc.wantErr {
				r.Error(refreshErr)
			} else {
				r.NoError(refreshErr)
				r.Equal(expectedUserId, refreshClaims.UserID)
				r.Equal(expectedRoleId, refreshClaims.RoleId)
			}
		})
	}
}
