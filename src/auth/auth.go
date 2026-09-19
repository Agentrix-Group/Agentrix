package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/golang-jwt/jwt/v5"
)

const (
	MinutesDuration = 60
	DaysDuration    = 7

	AccessTokenDuration  = MinutesDuration * time.Minute
	RefreshTokenDuration = DaysDuration * 24 * time.Hour
	TokenType            = "Bearer"
	Issuer               = "agentrix-api"
)

type Auth struct {
	accessSecret  []byte
	refreshSecret []byte
}

func NewAuth(accessSecret, refreshSecret string) *Auth {
	return &Auth{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
	}
}

func (a *Auth) GenerateAuthToken(participantId, roleId string) (*model.Token, error) {
	now := time.Now()
	accessExpiry := now.Add(AccessTokenDuration)
	refreshExpiry := now.Add(RefreshTokenDuration)

	accessToken, err := a.generateToken(participantId, roleId, accessExpiry, a.accessSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}

	refreshToken, err := a.generateToken(participantId, roleId, refreshExpiry, a.refreshSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %v", err)
	}

	return &model.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(AccessTokenDuration.Seconds()),
		TokenType:    TokenType,
	}, nil
}

func (a *Auth) generateToken(participantId, roleId string, expiry time.Time, secret []byte) (string, error) {
	claims := model.Claims{
		ParticipantId: participantId,
		RoleId:        roleId,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   participantId,
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func (a *Auth) ValidateAccessToken(tokenString string) (*model.Claims, error) {
	return a.validateToken(tokenString, a.accessSecret)
}

func (a *Auth) ValidateRefreshToken(tokenString string) (*model.Claims, error) {
	return a.validateToken(tokenString, a.refreshSecret)
}

func (a *Auth) validateToken(tokenString string, secret []byte) (*model.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(*model.Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
