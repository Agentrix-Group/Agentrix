// Command smoke verifies a running Agentrix stack end to end through the
// HTTP API: readiness with real components, login of admin and player,
// negative RBAC, agent creation, upload and sandboxed admission, enrollment
// locked to an exact submission, registration close, scheduling, idempotent
// run, polling with timeout, results bound to the committed run, replay
// digest and stream, ranking correctness and absence of orphan jobs/runs.
// It exits non-zero on the first failed check.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Agentrix-Group/Agentrix/cmd/internal/demo"
)

type checker struct{ n int }

func (c *checker) ok(format string, args ...any) {
	c.n++
	fmt.Printf("  [%02d] ok  %s\n", c.n, fmt.Sprintf(format, args...))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "smoke FAILED: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("smoke PASSED")
}

func run() error {
	api := envOr("AGENTRIX_API_URL", "http://localhost:8080")
	adminUser, adminPass := envOr("DEMO_ADMIN_USERNAME", "admin"), os.Getenv("DEMO_ADMIN_PASSWORD")
	playerUser, playerPass := envOr("DEMO_PLAYER_USERNAME", "player"), os.Getenv("DEMO_PLAYER_PASSWORD")
	if adminPass == "" || playerPass == "" {
		return fmt.Errorf("DEMO_ADMIN_PASSWORD and DEMO_PLAYER_PASSWORD are required")
	}
	botsDir := envOr("DEMO_BOTS_DIR", "games/starfighter/examples")
	c := &checker{}
	fmt.Printf("smoke against %s\n", api)

	anon := demo.New(api)
	if err := demo.Poll(2*time.Minute, "readiness", func() (bool, error) {
		res, err := anon.Do("GET", "/health/ready", nil)
		return err == nil && res.Status == http.StatusOK && res.JSON()["status"] != "down", nil
	}); err != nil {
		return err
	}
	c.ok("migrations applied and API ready (/health/ready)")

	admin, player := demo.New(api), demo.New(api)
	if err := admin.Login(adminUser, adminPass); err != nil {
		return err
	}
	if err := player.Login(playerUser, playerPass); err != nil {
		return err
	}
	c.ok("login admin and player (refresh token only in HttpOnly cookie)")

	readiness, err := admin.Expect(http.StatusOK, "GET", "/api/v1/admin/readiness", nil)
	if err != nil {
		return err
	}
	for _, comp := range readiness["components"].([]any) {
		m := comp.(map[string]any)
		if m["name"] == "database" || m["name"] == "workers" || m["name"] == "artifacts" {
			if m["status"] != "healthy" {
				return fmt.Errorf("component %v is %v: %v", m["name"], m["status"], m["message"])
			}
		}
	}
	c.ok("readiness reports healthy database, artifacts and live worker")

	for _, probe := range []struct{ method, path string }{
		{"GET", "/api/v1/admin/readiness"}, {"GET", "/api/v1/users"}, {"POST", "/api/v1/contests"},
	} {
		res, err := player.Do(probe.method, probe.path, map[string]any{})
		if err != nil {
			return err
		}
		if res.Status != http.StatusForbidden {
			return fmt.Errorf("RBAC: player got %d on %s %s", res.Status, probe.method, probe.path)
		}
	}
	if res, err := anon.Do("POST", "/api/v1/agents", map[string]string{"game_id": "starfighter", "name": "x"}); err != nil || res.Status != http.StatusUnauthorized {
		return fmt.Errorf("RBAC: anonymous agent creation not rejected (%d)", res.Status)
	}
	c.ok("negative RBAC: player forbidden on admin routes, anonymous cannot write")

	scenario := &demo.Scenario{Organizer: admin, Player: player, BotsDir: botsDir,
		ContestName: "Smoke " + time.Now().UTC().Format("20060102-150405"), Timeout: 5 * time.Minute,
		Log: func(format string, args ...any) { fmt.Printf("       %s\n", fmt.Sprintf(format, args...)) }}
	if err := scenario.Play(); err != nil {
		return err
	}
	c.ok("agents created, bundles uploaded and admitted in the sandbox")
	c.ok("enrollment locked to exact submissions; registration closed")
	c.ok("scheduling returned real run/job ids; idempotent replay returned the same run")
	c.ok("match finished within the polling timeout")

	match, err := admin.Expect(http.StatusOK, "GET", "/api/v1/matches/"+scenario.MatchID, nil)
	if err != nil {
		return err
	}
	if match["committed_run_id"] != scenario.RunID {
		return fmt.Errorf("committed run %v != scheduled run %s", match["committed_run_id"], scenario.RunID)
	}
	if n := len(match["results"].([]any)); n != 2 {
		return fmt.Errorf("expected 2 results for the committed run, got %d", n)
	}
	c.ok("results bound to the committed run %s", scenario.RunID)

	var replay map[string]any
	if err := demo.Poll(2*time.Minute, "replay publication", func() (bool, error) {
		m, err := admin.Expect(http.StatusOK, "GET", "/api/v1/matches/"+scenario.MatchID, nil)
		if err != nil {
			return false, err
		}
		replay, _ = m["replay"].(map[string]any)
		return replay != nil && replay["available"] == true, nil
	}); err != nil {
		return err
	}
	req, err := http.NewRequest("GET", api+"/api/v1/replays/"+replay["id"].(string)+"/stream", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept-Encoding", "gzip")
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if res.StatusCode != http.StatusOK || hex.EncodeToString(sum[:]) != replay["sha256"] {
		return fmt.Errorf("replay stream digest mismatch (status %d)", res.StatusCode)
	}
	c.ok("replay published; streamed bytes match sha256 %s", replay["sha256"])

	rankings, err := anon.Expect(http.StatusOK, "GET", "/api/v1/contests/"+scenario.ContestID+"/rankings", nil)
	if err != nil {
		return err
	}
	if err := demo.Poll(time.Minute, "ranking refresh", func() (bool, error) {
		rankings, err = anon.Expect(http.StatusOK, "GET", "/api/v1/contests/"+scenario.ContestID+"/rankings", nil)
		return err == nil && rankings["stale"] == false && rankings["applied_runs_count"] == float64(1), err
	}); err != nil {
		return err
	}
	rows := rankings["rankings"].([]any)
	points := 0.0
	for _, r := range rows {
		row := r.(map[string]any)
		if row["matches_played"] != float64(1) {
			return fmt.Errorf("ranking row %v played %v matches", row["agent_name"], row["matches_played"])
		}
		points += row["points"].(float64)
	}
	if len(rows) != 2 || (points != 3 && points != 2) {
		return fmt.Errorf("unexpected ranking: %v", rows)
	}
	c.ok("ranking projects exactly one committed run (%d rows, %v points)", len(rows), points)

	readiness, err = admin.Expect(http.StatusOK, "GET", "/api/v1/admin/readiness", nil)
	if err != nil {
		return err
	}
	for _, comp := range readiness["components"].([]any) {
		m := comp.(map[string]any)
		if m["name"] == "integrity" && m["status"] != "healthy" {
			return fmt.Errorf("integrity: %v", m["details"])
		}
		if m["name"] == "queue" && m["details"].(map[string]any)["expired_leases"] != float64(0) {
			return fmt.Errorf("queue has expired leases: %v", m["details"])
		}
	}
	c.ok("no orphan jobs or runs; no expired leases")
	return nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
