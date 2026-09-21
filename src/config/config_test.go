package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func setenv(t *testing.T, kv map[string]string) {
	t.Helper()
	for _, k := range []string{"MODE", "ACCESS_SECRET", "CORS_ALLOWED_ORIGINS", "TRUSTED_PROXIES", "AGENTRIX_SANDBOX", "COOKIE_SECURE",
		"COOKIE_SAMESITE", "ACCESS_TOKEN_TTL"} {
		t.Setenv(k, "")
	}
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

const strong = "0123456789abcdef0123456789abcdef-strong"

func TestDevModeUsesEphemeralSecret(t *testing.T) {
	setenv(t, map[string]string{"MODE": "dev"})
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.Auth.EphemeralSecret)
	require.Len(t, cfg.Auth.AccessSecret, 48)
}

func TestSecretsAreMandatoryOutsideDev(t *testing.T) {
	for _, mode := range []string{"demo", "production"} {
		setenv(t, map[string]string{"MODE": mode, "COOKIE_SECURE": "true"})
		_, err := Load()
		require.ErrorContains(t, err, "ACCESS_SECRET is required", mode)
	}
	setenv(t, map[string]string{"MODE": "production", "ACCESS_SECRET": "short", "COOKIE_SECURE": "true"})
	_, err := Load()
	require.ErrorContains(t, err, "at least 32 bytes")
	setenv(t, map[string]string{"MODE": "production", "ACCESS_SECRET": "agentrix-access-secret-key-change-in-prod", "COOKIE_SECURE": "true"})
	_, err = Load()
	require.ErrorContains(t, err, "placeholder")
}

func TestProductionRequiresSecureCookies(t *testing.T) {
	setenv(t, map[string]string{"MODE": "production", "ACCESS_SECRET": strong, "COOKIE_SECURE": "false"})
	_, err := Load()
	require.ErrorContains(t, err, "COOKIE_SECURE")
	setenv(t, map[string]string{"MODE": "production", "ACCESS_SECRET": strong})
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.Auth.CookieSecure)
}

func TestCORSWildcardIsRejected(t *testing.T) {
	setenv(t, map[string]string{"MODE": "dev", "CORS_ALLOWED_ORIGINS": "http://localhost:5173,*"})
	_, err := Load()
	require.ErrorContains(t, err, "cannot contain '*'")
	setenv(t, map[string]string{"MODE": "dev", "CORS_ALLOWED_ORIGINS": "localhost:5173"})
	_, err = Load()
	require.ErrorContains(t, err, "invalid CORS origin")
}

func TestDirectSandboxOnlyInDev(t *testing.T) {
	setenv(t, map[string]string{"MODE": "demo", "ACCESS_SECRET": strong, "AGENTRIX_SANDBOX": "direct"})
	_, err := Load()
	require.ErrorContains(t, err, "only allowed in dev/test")
	setenv(t, map[string]string{"MODE": "test", "AGENTRIX_SANDBOX": "direct"})
	_, err = Load()
	require.NoError(t, err)
}

func TestTrustedProxiesAndDSNEscaping(t *testing.T) {
	setenv(t, map[string]string{"MODE": "dev", "TRUSTED_PROXIES": "10.0.0.0/8,127.0.0.1"})
	t.Setenv("DB_PASSWORD", "p@ss:w/rd")
	cfg, err := Load()
	require.NoError(t, err)
	require.Len(t, cfg.HTTP.TrustedProxies, 2)
	require.True(t, strings.Contains(cfg.DSN(), "p%40ss%3Aw%2Frd"), cfg.DSN())
	setenv(t, map[string]string{"MODE": "dev", "TRUSTED_PROXIES": "not-an-ip"})
	_, err = Load()
	require.Error(t, err)
}

func TestAccessTokenTTLIsBounded(t *testing.T) {
	setenv(t, map[string]string{"MODE": "dev", "ACCESS_TOKEN_TTL": "2h"})
	_, err := Load()
	require.ErrorContains(t, err, "must not exceed 30m")
}
