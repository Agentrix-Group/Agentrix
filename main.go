package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/F4nk1/Agentrix/src/config"
	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/executor"
	"github.com/F4nk1/Agentrix/src/game"
	"github.com/F4nk1/Agentrix/src/repository"
	"github.com/F4nk1/Agentrix/src/server"
	"github.com/F4nk1/Agentrix/src/service"
	"github.com/F4nk1/Agentrix/src/tracer"
)

func main() {
	ctx := context.Background()
	cfg := config.NewConfiguration()
	tracer.Infof(ctx, "Starting Agentrix in '%s' mode, database host '%s:%s'", cfg.Mode, cfg.Database.Host, cfg.Database.Port)

	// Initialize Artifact Store
	artifacts, err := connection.NewArtifactStore(ctx, cfg)
	if err != nil {
		tracer.Fatalf(ctx, "Failed to initialize artifact store: %s", err)
	}

	// Initialize Match Job Queue
	queue := connection.NewJobQueue(100)
	defer queue.Close()

	// Load Game Registry
	registry := game.GetRegistry()
	if err := registry.LoadGamesFromDir("./games"); err != nil {
		tracer.Warnf(ctx, "Could not load games directory: %s", err)
	}

	// Initialize Database Connection
	conn, err := connection.NewConnection(ctx, cfg)
	if err != nil {
		tracer.Errorf(ctx, "Warning: Database connection failed (%s). Continuing in degraded mode for local tests.", err)
	} else {
		defer conn.Close()
	}

	// Compose Layers
	repo := repository.NewRepository(conn)
	svc := service.NewService(repo, artifacts, queue)

	// Setup Sandbox & Match Executor Worker Pool
	sandbox := executor.NewSandbox(0)
	matchExecutor := executor.NewMatchExecutor(svc, sandbox)
	workerPool := executor.NewWorkerPool(queue, matchExecutor, 2)
	workerPool.Start(ctx)
	defer workerPool.Stop()

	// Initialize HTTP Server
	srv := server.NewServer(svc)

	serverHost := fmt.Sprintf(":%s", cfg.Server.Port)
	tracer.Infof(ctx, "Starting Agentrix server on %s", serverHost)
	tracer.Fatal(ctx, http.ListenAndServe(serverHost, srv.Handler))
}
