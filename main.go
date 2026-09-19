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

	// Load Game Registry
	registry := game.GetRegistry()
	if err := registry.LoadGamesFromDir("./games"); err != nil {
		tracer.FatalEvent(ctx, tracer.ScopeSystem, "starfighter.unavailable", "No se pudo cargar el manifiesto de Starfighter",
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

	// PostgreSQL is the authoritative queue when available. Its reservation
	// query uses FOR UPDATE SKIP LOCKED; the memory queue is only degraded mode.
	var queue connection.JobQueue
	if conn != nil && conn.Db != nil {
		queue, err = connection.NewPostgresJobQueue(conn.Db, "")
		if err != nil {
			tracer.WarnEvent(ctx, tracer.ScopeQueue, "queue.degraded", "No se pudo preparar la cola PostgreSQL",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
	}
	if queue == nil {
		queue = connection.NewJobQueue(100)
	}
	defer queue.Close()

	// Compose Layers
	repo := repository.NewRepository(conn)
	sandbox := executor.NewSandbox(0)
	svc := service.NewService(repo, artifacts, queue, sandbox)

	// Setup Match Executor Worker Pool
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
