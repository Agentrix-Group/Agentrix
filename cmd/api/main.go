// Command api serves the Agentrix HTTP API. It never executes bots or the
// engine and refuses to start against a database that is not the canonical
// baseline.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/server"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "agentrix-api: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration:\n%w", err)
	}
	tracer.Configure(tracer.Config{Level: cfg.Logging.Level, Format: cfg.Logging.Format, Color: cfg.Logging.Color})
	defer func() { _ = tracer.Sync() }()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if cfg.Auth.EphemeralSecret {
		tracer.WarnEvent(ctx, tracer.ScopeSystem, "auth.ephemeral_secret",
			"ACCESS_SECRET not set: using a random per-process secret (dev/test only)")
	}

	svc, db, err := buildService(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := svc.SyncGames(ctx); err != nil {
		return fmt.Errorf("register games: %w", err)
	}

	srv := server.New(svc, cfg)
	defer srv.Close()
	httpServer := &http.Server{Addr: ":" + cfg.HTTP.Port, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout: 60 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 120 * time.Second}
	errCh := make(chan error, 1)
	go func() {
		tracer.InfoEvent(ctx, tracer.ScopeHTTP, "http.ready", "API listening", tracer.String("address", httpServer.Addr),
			tracer.String("mode", cfg.Mode))
		errCh <- httpServer.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}

func buildService(ctx context.Context, cfg *config.Config) (*service.Service, interface{ Close() error }, error) {
	games, err := game.LoadRegistry(cfg.GamesDir)
	if err != nil {
		return nil, nil, fmt.Errorf("load games: %w", err)
	}
	artifacts, err := connection.NewArtifactStore(cfg.ArtifactsDir)
	if err != nil {
		return nil, nil, err
	}
	db, err := connection.OpenDatabase(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	if err := database.CheckSchemaCompatible(ctx, db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("database schema: %w", err)
	}
	tokens, err := auth.NewTokens(cfg.Auth.AccessSecret, cfg.Auth.AccessTTL)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	opts := service.DefaultOptions()
	opts.RefreshIdleTTL, opts.SessionMaxTTL, opts.MaxSessions = cfg.Auth.RefreshIdleTTL, cfg.Auth.SessionMaxTTL, cfg.Auth.MaxSessions
	opts.RegistrationOpen = cfg.Auth.RegistrationOpen
	opts.LeaseTTL, opts.MaxRunAttempts = cfg.Worker.LeaseTTL, cfg.Worker.MaxAttempts
	return service.New(repository.NewStore(db), games, artifacts, tokens, opts), db, nil
}

// healthcheck probes the local liveness endpoint (container HEALTHCHECK in
// a distroless image without shell or curl).
func healthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	client := http.Client{Timeout: 2 * time.Second}
	res, err := client.Get("http://127.0.0.1:" + port + "/health/live")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
