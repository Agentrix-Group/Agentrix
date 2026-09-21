// Command bootstrap prepares a demo environment. The first administrator is
// created through the BootstrapAdmin use case (the only step that needs
// database access, since no account exists yet); everything else goes
// through the public HTTP API exactly as a user would: accounts, agents,
// uploads, admission by the worker, enrollment, contest transitions, match,
// idempotent run, results, replay and a published ranking snapshot.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Agentrix-Group/Agentrix/cmd/internal/demo"
	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
)

const contestName = "Agentrix Demo Cup"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap: %v\n", err)
		os.Exit(1)
	}
}

func required(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return v, nil
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	adminUser := envOr("DEMO_ADMIN_USERNAME", "admin")
	playerUser := envOr("DEMO_PLAYER_USERNAME", "player")
	adminPass, err := required("DEMO_ADMIN_PASSWORD")
	if err != nil {
		return err
	}
	playerPass, err := required("DEMO_PLAYER_PASSWORD")
	if err != nil {
		return err
	}
	apiURL := envOr("AGENTRIX_API_URL", "http://localhost:8080")
	botsDir := envOr("DEMO_BOTS_DIR", cfg.GamesDir+"/starfighter/examples")
	logf := func(format string, args ...any) { fmt.Printf("bootstrap: "+format+"\n", args...) }

	if err := ensureAdmin(cfg, adminUser, adminPass); err != nil {
		return err
	}
	admin := demo.New(apiURL)
	if err := demo.Poll(2*time.Minute, "API readiness", func() (bool, error) {
		res, err := admin.Do("GET", "/health/ready", nil)
		return err == nil && res.Status == http.StatusOK, nil
	}); err != nil {
		return err
	}
	if err := admin.Login(adminUser, adminPass); err != nil {
		return err
	}
	contests, err := admin.Expect(http.StatusOK, "GET", "/api/v1/contests", nil)
	if err != nil {
		return err
	}
	for _, c := range contests["items"].([]any) {
		if c.(map[string]any)["name"] == contestName {
			logf("demo already bootstrapped (contest %s exists); nothing to do", c.(map[string]any)["id"])
			return nil
		}
	}

	res, err := admin.Do("POST", "/api/v1/users", map[string]any{"username": playerUser, "email": playerUser + "@demo.agentrix.local",
		"password": playerPass, "roles": []string{model.RolePlayer}})
	if err != nil {
		return err
	}
	switch {
	case res.Status == http.StatusCreated:
		logf("created player %s", playerUser)
	case res.Status == http.StatusConflict && res.ErrorCode() == "user_exists":
		logf("player %s already exists", playerUser)
	default:
		return fmt.Errorf("create player: %d %s", res.Status, res.Body)
	}
	player := demo.New(apiURL)
	if err := player.Login(playerUser, playerPass); err != nil {
		return err
	}

	scenario := &demo.Scenario{Organizer: admin, Player: player, BotsDir: botsDir, ContestName: contestName,
		Timeout: 5 * time.Minute, Log: logf}
	if err := scenario.Play(); err != nil {
		return err
	}
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
	snapshot, err := admin.Expect(http.StatusCreated, "POST", "/api/v1/contests/"+scenario.ContestID+"/rankings/snapshots", nil)
	if err != nil {
		return err
	}
	logf("match %s finished; replay %s published; ranking snapshot v%v", scenario.MatchID, replay["id"], snapshot["version"])
	if cfg.Mode == config.ModeDemo || cfg.Development() {
		logf("demo credentials (demo mode only): admin=%s player=%s (passwords from DEMO_*_PASSWORD)", adminUser, playerUser)
	}
	return nil
}

func ensureAdmin(cfg *config.Config, username, password string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := connection.OpenDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.CheckSchemaCompatible(ctx, db); err != nil {
		return err
	}
	games, err := game.LoadRegistry(cfg.GamesDir)
	if err != nil {
		return err
	}
	artifacts, err := connection.NewArtifactStore(cfg.ArtifactsDir)
	if err != nil {
		return err
	}
	tokens, err := auth.NewTokens([]byte("bootstrap-does-not-issue-access-tokens"), time.Minute)
	if err != nil {
		return err
	}
	svc := service.New(repository.NewStore(db), games, artifacts, tokens, service.DefaultOptions())
	_, err = svc.BootstrapAdmin(ctx, username, username+"@demo.agentrix.local", password)
	if domain, ok := model.AsError(err); ok && domain.Code == "admin_exists" {
		fmt.Println("bootstrap: administrator already exists")
		return nil
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("bootstrap admin: %w", err)
	}
	fmt.Printf("bootstrap: created administrator %s\n", username)
	return nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
