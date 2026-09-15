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
	tracer.Configure(tracer.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Color:  cfg.Logging.Color,
	})
	defer tracer.Sync()
	tracer.InfoEvent(ctx, tracer.ScopeSystem, "system.starting", "Agentrix iniciando",
		tracer.String("mode", cfg.Mode),
	)

	// Initialize Artifact Store
	artifacts, err := connection.NewArtifactStore(ctx, cfg)
	if err != nil {
		tracer.FatalEvent(ctx, tracer.ScopeArtifact, "artifact.unavailable", "No se pudo preparar el almacén de artefactos",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	}
	tracer.InfoEvent(ctx, tracer.ScopeArtifact, "artifact.ready", "Almacén de artefactos disponible")

	// Initialize Match Job Queue
	queue := connection.NewJobQueue(100)
	defer queue.Close()

	// Load Game Registry
	registry := game.GetRegistry()
	if err := registry.LoadGamesFromDir("./games"); err != nil {
		tracer.WarnEvent(ctx, tracer.ScopeSystem, "games.degraded", "Algunos juegos no pudieron cargarse",
			tracer.Origin(tracer.OriginGame), tracer.Err(err))
	}

	// Initialize Database Connection
	conn, err := connection.NewConnection(ctx, cfg)
	if err != nil {
		tracer.WarnEvent(ctx, tracer.ScopeDatabase, "database.unavailable", "Sin conexión; Agentrix continúa en modo degradado",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	} else {
		defer conn.Close()
		tracer.InfoEvent(ctx, tracer.ScopeDatabase, "database.ready", "Base de datos disponible")
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
	tracer.InfoEvent(ctx, tracer.ScopeHTTP, "http.ready", "Servidor disponible", tracer.String("address", serverHost))
	if err := http.ListenAndServe(serverHost, srv.Handler); err != nil {
		tracer.FatalEvent(ctx, tracer.ScopeHTTP, "http.stopped", "El servidor se detuvo",
			tracer.Origin(tracer.OriginPlatform), tracer.Err(err))
	}
}
