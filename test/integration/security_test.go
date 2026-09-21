package integration

import (
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/server"
	"github.com/stretchr/testify/require"
)

// P0-12: a player cannot modify, read or disable another player's agent.
func TestSecurity_NoHorizontalAccessToAgents(t *testing.T) {
	e := newEnv(t, envOptions{})
	alice := e.user("alice", model.RolePlayer)
	bob := e.user("bob", model.RolePlayer)
	bobAgent := bob.expect(http.StatusCreated, "POST", "/api/v1/agents", map[string]string{"game_id": "starfighter", "name": "Bob"})
	id := bobAgent["id"].(string)

	for _, req := range []struct {
		method, path string
		body         any
	}{
		{"PATCH", "/api/v1/agents/" + id, map[string]string{"name": "pwned"}},
		{"PUT", "/api/v1/agents/" + id + "/status", map[string]string{"status": "disabled"}},
		{"GET", "/api/v1/agents/" + id, nil},
		{"GET", "/api/v1/agents/" + id + "/submissions", nil},
		{"POST", "/api/v1/agents/" + id + "/submissions", bundleUpload(t, "x", []byte("print(1)\n"))},
	} {
		res := alice.do(req.method, req.path, req.body)
		require.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, res.Status, "%s %s -> %d %s", req.method, req.path, res.Status, res.Body)
	}
	got := bob.expect(http.StatusOK, "GET", "/api/v1/agents/"+id, nil)
	require.Equal(t, "Bob", got["name"])
	require.Equal(t, "active", got["status"])
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM submissions`))

	// Listing returns only own agents unless agents:read:any is held.
	require.Len(t, alice.expect(http.StatusOK, "GET", "/api/v1/agents", nil)["items"], 0)
	require.Equal(t, http.StatusForbidden, alice.do("GET", "/api/v1/agents?owner=*", nil).Status)
	org := e.user("org", model.RoleOrganizer)
	require.Len(t, org.expect(http.StatusOK, "GET", "/api/v1/agents?owner=*", nil)["items"], 1)
	require.Equal(t, http.StatusForbidden, org.do("PATCH", "/api/v1/agents/"+id, map[string]string{"name": "x"}).Status)
}

// P0-11: replacing roles removes the previous ones for every live session.
func TestSecurity_RoleRevocationIsImmediate(t *testing.T) {
	e := newEnv(t, envOptions{})
	root := e.client()
	e.adminID()
	root.login("root", "password-root")
	carol := e.user("carol", model.RoleAdmin)
	carol.expect(http.StatusOK, "GET", "/api/v1/admin/readiness", nil)

	root.expect(http.StatusOK, "PUT", "/api/v1/users/"+carol.id()+"/roles", map[string]any{"roles": []string{"player"}})
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM user_roles WHERE user_id = $1 AND role_id = 'admin'`, carol.id()))
	// Same access token, next request: admin capability is gone.
	require.Equal(t, http.StatusForbidden, carol.do("GET", "/api/v1/admin/readiness", nil).Status)
	me := carol.expect(http.StatusOK, "GET", "/api/v1/me", nil)
	require.Equal(t, []any{"player"}, me["roles"])
	require.NotContains(t, me["capabilities"], "admin:access")

	// Administrators cannot demote themselves (lockout protection).
	res := root.do("PUT", "/api/v1/users/"+root.id()+"/roles", map[string]any{"roles": []string{"player"}})
	require.Equal(t, http.StatusConflict, res.Status)
	// Unknown and legacy roles are rejected.
	res = root.do("PUT", "/api/v1/users/"+carol.id()+"/roles", map[string]any{"roles": []string{"participant"}})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status)
}

// Disabling an account revokes its sessions; the status field is mandatory.
func TestSecurity_UserStatusIsExplicitAndRevokesSessions(t *testing.T) {
	e := newEnv(t, envOptions{})
	root := e.client()
	e.adminID()
	root.login("root", "password-root")
	dave := e.user("dave", model.RolePlayer)

	res := root.do("PUT", "/api/v1/users/"+dave.id()+"/status", map[string]any{})
	require.Equal(t, http.StatusUnprocessableEntity, res.Status, "missing status must never mean disabled")
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM users WHERE id = $1 AND status = 'active'`, dave.id()))

	root.expect(http.StatusOK, "PUT", "/api/v1/users/"+dave.id()+"/status", map[string]string{"status": "suspended"})
	require.Equal(t, http.StatusUnauthorized, dave.do("GET", "/api/v1/me", nil).Status)
	require.Equal(t, http.StatusForbidden, e.client().do("POST", "/api/v1/auth/login", map[string]string{"username": "dave", "password": "password-dave"}).Status)

	// Inactive accounts remain listable and can be reactivated.
	listed := root.expect(http.StatusOK, "GET", "/api/v1/users?status=suspended", nil)["items"].([]any)
	require.Len(t, listed, 1)
	root.expect(http.StatusOK, "PUT", "/api/v1/users/"+dave.id()+"/status", map[string]string{"status": "active"})
	dave.login("dave", "password-dave")
	dave.expect(http.StatusOK, "GET", "/api/v1/me", nil)
}

// P0-13: refresh rotation is transactional; concurrent refreshes of the same
// token never mint two successors, and reuse revokes the family.
func TestSecurity_RefreshRotationIsAtomic(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.user("erin", model.RolePlayer)
	for i := 0; i < 15; i++ {
		c := e.client()
		c.login("erin", "password-erin")
		var wg sync.WaitGroup
		statuses := make([]int, 2)
		for j := 0; j < 2; j++ {
			wg.Add(1)
			go func(j int) {
				defer wg.Done()
				statuses[j] = c.do("POST", "/api/v1/auth/refresh", nil, server.CSRFHeader, server.CSRFHeaderValue).Status
			}(j)
		}
		wg.Wait()
		ok := 0
		for _, s := range statuses {
			if s == http.StatusOK {
				ok++
			}
		}
		require.LessOrEqual(t, ok, 1, "iteration %d: statuses %v", i, statuses)
	}
	// Every family has at most one unused token.
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM (SELECT family_id FROM refresh_tokens WHERE used_at IS NULL
		GROUP BY family_id HAVING COUNT(*) > 1) x`))
}

func TestSecurity_SessionLifecycle(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.user("frank", model.RolePlayer)
	c := e.client()
	c.login("frank", "password-frank")

	// The refresh token is only an HttpOnly cookie, never in the body.
	res := c.do("POST", "/api/v1/auth/login", map[string]string{"username": "frank", "password": "password-frank"})
	require.Equal(t, http.StatusOK, res.Status)
	require.NotContains(t, string(res.Body), "artx_rf_")
	setCookie := res.Header.Get("Set-Cookie")
	require.Contains(t, setCookie, server.RefreshCookieName+"=artx_rf_")
	require.Contains(t, setCookie, "HttpOnly")
	require.Contains(t, setCookie, "Path=/api/v1/auth")
	require.Contains(t, setCookie, "SameSite=Strict")

	// Cookie endpoints require the CSRF header.
	require.Equal(t, http.StatusForbidden, c.do("POST", "/api/v1/auth/refresh", nil).Status)
	refreshed := c.expect(http.StatusOK, "POST", "/api/v1/auth/refresh", nil, server.CSRFHeader, server.CSRFHeaderValue)
	c.token = refreshed["access_token"].(string)
	c.expect(http.StatusOK, "GET", "/api/v1/me", nil)

	// Logout revokes the family: the access token of that session dies too.
	c.expect(http.StatusNoContent, "POST", "/api/v1/auth/logout", nil, server.CSRFHeader, server.CSRFHeaderValue)
	require.Equal(t, http.StatusUnauthorized, c.do("GET", "/api/v1/me", nil).Status)
	require.Equal(t, http.StatusUnauthorized, c.do("POST", "/api/v1/auth/refresh", nil, server.CSRFHeader, server.CSRFHeaderValue).Status)

	// logout-all revokes every family of the user.
	a, b := e.client(), e.client()
	a.login("frank", "password-frank")
	b.login("frank", "password-frank")
	a.expect(http.StatusOK, "POST", "/api/v1/auth/logout-all", nil)
	require.Equal(t, http.StatusUnauthorized, b.do("GET", "/api/v1/me", nil).Status)
}

func TestSecurity_ReusedRefreshTokenRevokesFamily(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.user("gina", model.RolePlayer)
	c := e.client()
	c.login("gina", "password-gina")
	var stolen string
	for _, ck := range c.http.Jar.Cookies(mustURL(e.http.URL + "/api/v1/auth")) {
		if ck.Name == server.RefreshCookieName {
			stolen = ck.Value
		}
	}
	require.NotEmpty(t, stolen)
	refreshed := c.expect(http.StatusOK, "POST", "/api/v1/auth/refresh", nil, server.CSRFHeader, server.CSRFHeaderValue)
	c.token = refreshed["access_token"].(string)

	attacker := e.client()
	attacker.http.Jar.SetCookies(mustURL(e.http.URL+"/api/v1/auth"), []*http.Cookie{{Name: server.RefreshCookieName, Value: stolen, Path: "/api/v1/auth"}})
	res := attacker.do("POST", "/api/v1/auth/refresh", nil, server.CSRFHeader, server.CSRFHeaderValue)
	require.Equal(t, http.StatusUnauthorized, res.Status)
	require.Equal(t, "refresh_token_reused", res.ErrorCode(t))
	// The legitimate session of that family is revoked as well.
	require.Equal(t, http.StatusUnauthorized, c.do("GET", "/api/v1/me", nil).Status)
}

func TestSecurity_TokensAndErrors(t *testing.T) {
	e := newEnv(t, envOptions{})
	anon := e.client()
	// Invalid bearer is rejected, never downgraded to anonymous.
	anon.token = "not-a-jwt"
	require.Equal(t, http.StatusUnauthorized, anon.do("GET", "/api/v1/contests", nil).Status)
	// alg=none and foreign-signed tokens are rejected.
	anon.token = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJ4Iiwic2lkIjoieSJ9."
	require.Equal(t, http.StatusUnauthorized, anon.do("GET", "/api/v1/contests", nil).Status)
	anon.token = ""
	// Anonymous visitors get spectator capabilities only.
	anon.expect(http.StatusOK, "GET", "/api/v1/contests", nil)
	require.Equal(t, http.StatusUnauthorized, anon.do("POST", "/api/v1/agents", map[string]string{"game_id": "starfighter", "name": "x"}).Status)
	// Unknown routes are 404 with the stable error envelope.
	res := anon.do("GET", "/api/v1/nope", nil)
	require.Equal(t, http.StatusNotFound, res.Status)
	require.Equal(t, "route_not_found", res.ErrorCode(t))
	// Unknown JSON fields are rejected instead of ignored.
	res = anon.do("POST", "/api/v1/auth/login", map[string]any{"username": "x", "password": "y", "role_id": "admin"})
	require.Equal(t, http.StatusBadRequest, res.Status)
	// Errors never leak SQL.
	require.False(t, strings.Contains(strings.ToLower(string(res.Body)), "sql"))
}

func TestSecurity_CORSRejectsUnknownOrigins(t *testing.T) {
	e := newEnv(t, envOptions{})
	c := e.client()
	res := c.do("OPTIONS", "/api/v1/contests", nil, "Origin", "https://evil.example", "Access-Control-Request-Method", "GET")
	require.Equal(t, http.StatusForbidden, res.Status)
	res = c.do("OPTIONS", "/api/v1/contests", nil, "Origin", "http://localhost:5173", "Access-Control-Request-Method", "GET")
	require.Equal(t, http.StatusNoContent, res.Status)
	require.Equal(t, "http://localhost:5173", res.Header.Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", res.Header.Get("Access-Control-Allow-Credentials"))
}
