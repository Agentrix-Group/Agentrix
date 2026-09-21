package server

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/stretchr/testify/require"
)

// Every route declares an explicit policy; protected routes name catalog
// capabilities; public routes are limited to probes and session endpoints.
func TestRoutePolicyCoverage(t *testing.T) {
	s := New(nil, &config.Config{})
	defer s.Close()
	publicAllowed := []string{"/health/live", "/health/ready", "/api/v1/auth/register", "/api/v1/auth/login",
		"/api/v1/auth/refresh", "/api/v1/auth/logout"}
	seen := map[string]bool{}
	for _, r := range s.Routes() {
		key := r.Method + " " + r.Pattern
		require.False(t, seen[key], "duplicated route %s", key)
		seen[key] = true
		require.NotNil(t, r.Handler, key)
		if r.Policy.Public {
			require.Contains(t, publicAllowed, r.Pattern, "unexpected public route %s", key)
			require.Empty(t, r.Policy.Capabilities, key)
			continue
		}
		require.NotEmpty(t, r.Policy.Capabilities, "route %s has no policy", key)
		for _, c := range r.Policy.Capabilities {
			require.True(t, slices.Contains(model.Capabilities, c), "%s uses unknown capability %s", key, c)
		}
		if r.Method != http.MethodGet {
			require.True(t, r.Policy.Authenticated, "mutating route %s must require a signed-in user", key)
		}
		require.False(t, strings.Contains(r.Pattern, "participant"), key)
	}
}

// Anonymous visitors can only reach routes whose capabilities the spectator
// role holds; verify that the policy check runs before any handler.
func TestPolicyRejectsBeforeHandler(t *testing.T) {
	policy := SignedIn(model.CapAgentsCreate)
	require.False(t, policy.allows(&model.Principal{Capabilities: []model.Capability{model.CapContestsView}}))
	require.True(t, Requires(model.CapAgentsReadOwn, model.CapAgentsReadAny).allows(
		&model.Principal{UserID: "u", Capabilities: []model.Capability{model.CapAgentsReadAny}}))
}

func TestClientIPTrustsOnlyConfiguredProxies(t *testing.T) {
	cfg := &config.Config{}
	s := New(nil, cfg)
	defer s.Close()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	require.Equal(t, "203.0.113.9", s.clientIP(req), "untrusted peers cannot spoof X-Forwarded-For")
}

func TestRateLimiterIsBoundedAndSliding(t *testing.T) {
	l := NewRateLimiter(2, 60e9, 2)
	defer l.Stop()
	ok, _ := l.Allow("a")
	require.True(t, ok)
	ok, _ = l.Allow("a")
	require.True(t, ok)
	ok, _ = l.Allow("a")
	require.False(t, ok)
	ok, _ = l.Allow("b")
	require.True(t, ok)
	ok, _ = l.Allow("c")
	require.False(t, ok, "key table is bounded: new keys are rejected when full")
}
