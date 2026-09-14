package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateToken(t *testing.T) {
	r := require.New(t)
	auth := NewAuth("test-access-secret-key", "test-refresh-secret-key")

	testCases := []struct {
		name          string
		participantId string
		roleId        string
	}{
		{name: "Valid inputs", participantId: "user123", roleId: "admin"},
		{name: "Participant inputs", participantId: "part-456", roleId: "participant"},
		{name: "Special characters", participantId: "user@domain.com", roleId: "referee"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := auth.GenerateAuthToken(tc.participantId, tc.roleId)
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

	expectedParticipantId := "user123"
	expectedRoleId := "admin"

	validToken, err := auth.GenerateAuthToken(expectedParticipantId, expectedRoleId)
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
				r.Equal(expectedParticipantId, accessClaims.ParticipantId)
				r.Equal(expectedRoleId, accessClaims.RoleId)
			}

			refreshClaims, refreshErr := auth.ValidateRefreshToken(tc.refreshToken)
			if tc.wantErr {
				r.Error(refreshErr)
			} else {
				r.NoError(refreshErr)
				r.Equal(expectedParticipantId, refreshClaims.ParticipantId)
				r.Equal(expectedRoleId, refreshClaims.RoleId)
			}
		})
	}
}
