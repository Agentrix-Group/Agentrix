package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.NewConfiguration()
	tracer.Configure(tracer.Config{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
		Color:  cfg.Logging.Color,
	})
	defer tracer.Sync()

	tracer.InfoEvent(ctx, tracer.ScopeWorker, "worker.starting", "Agentrix Worker iniciando",
		tracer.String("mode", cfg.Mode),
	)

	// Production Fail-Closed Validations
	if cfg.Mode != config.ModeDev {
		bwrap := &executor.BubblewrapRuntime{}
		podman := &executor.PodmanRuntime{}
		if !bwrap.IsAvailable() && !podman.IsAvailable() {
			tracer.FatalEvent(ctx, tracer.ScopeWorker, "sandbox.unavailable", "En producción, el worker requiere Bubblewrap o Podman instalado",
				tracer.Origin(tracer.OriginPlatform))
		}
	}

	// Initialize Artifact Store
	artifacts, err := connection.NewArtifactStore(ctx, cfg)
	if err != nil {
		tracer.FatalEvent(ctx, tracer.ScopeArtifact, "artifact.unavailable", "No se pudo preparar el almacén de artefactos",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	}

	// Load Game Registry
	registry := game.GetRegistry()
	if err := registry.LoadGamesFromDir("./games"); err != nil {
		tracer.FatalEvent(ctx, tracer.ScopeSystem, "starfighter.unavailable", "No se pudo cargar el manifiesto del juego",
			tracer.Origin(tracer.OriginGame), tracer.Err(err))
	}

	// Initialize Database Connection
	conn, err := connection.NewConnection(ctx, cfg)
	if err != nil {
		if cfg.Mode != config.ModeDev {
			tracer.FatalEvent(ctx, tracer.ScopeDatabase, "database.required", "En producción, el worker requiere PostgreSQL configurado y disponible",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
		tracer.WarnEvent(ctx, tracer.ScopeDatabase, "database.unavailable", "Sin conexión; worker continúa en modo degradado",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	} else {
		defer conn.Close()
		tracer.InfoEvent(ctx, tracer.ScopeDatabase, "database.ready", "Base de datos disponible")
	}

	// Authoritative Queue
	var queue connection.JobQueue
	if conn != nil && conn.Db != nil {
		queue, err = connection.NewPostgresJobQueue(conn.Db, "")
		if err != nil {
			tracer.WarnEvent(ctx, tracer.ScopeQueue, "queue.degraded", "No se pudo preparar la cola PostgreSQL",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
	}
	if queue == nil {
		if cfg.Mode != config.ModeDev {
			tracer.FatalEvent(ctx, tracer.ScopeQueue, "queue.required", "En producción, el worker requiere la cola autoritativa en PostgreSQL")
		}
		queue = connection.NewJobQueue(100)
	}
	defer queue.Close()

	// Compose Layers
	repo := repository.NewRepository(conn)
	sandbox := executor.NewSandbox(0)
	svc := service.NewService(repo, artifacts, queue, sandbox)

	// Setup Match Executor & Worker Pool
	matchExecutor := executor.NewMatchExecutor(svc, sandbox)
	workerPool := executor.NewWorkerPool(queue, matchExecutor, 2)
	workerPool.Start(ctx)
	tracer.InfoEvent(ctx, tracer.ScopeWorker, "worker.pool.started", "Worker pool activo procesando partidas")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	sig := <-sigChan
	tracer.InfoEvent(ctx, tracer.ScopeWorker, "worker.stopping", "Señal recibida, deteniendo worker pool...",
		tracer.String("signal", sig.String()))
	workerPool.Stop()
	tracer.InfoEvent(ctx, tracer.ScopeWorker, "worker.stopped", "Worker detenido limpiamente")
}
