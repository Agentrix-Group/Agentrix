package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/server"
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
	tracer.InfoEvent(ctx, tracer.ScopeSystem, "api.starting", "Agentrix API iniciando",
		tracer.String("mode", cfg.Mode),
	)

	// Initialize Artifact Store
	artifacts, err := connection.NewArtifactStore(ctx, cfg)
	if err != nil {
		tracer.FatalEvent(ctx, tracer.ScopeArtifact, "artifact.unavailable", "No se pudo preparar el almacén de artefactos",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	}

	// Initialize Database Connection
	conn, err := connection.NewConnection(ctx, cfg)
	if err != nil {
		if cfg.Mode != config.ModeDev {
			tracer.FatalEvent(ctx, tracer.ScopeDatabase, "database.required", "En producción, la API requiere PostgreSQL configurado y disponible",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
		tracer.WarnEvent(ctx, tracer.ScopeDatabase, "database.unavailable", "Sin conexión; API continúa en modo degradado",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
	} else {
		defer conn.Close()
		tracer.InfoEvent(ctx, tracer.ScopeDatabase, "database.ready", "Base de datos disponible")
	}

	// Authoritative PostgreSQL Queue (or in-memory degraded queue in dev)
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
			tracer.FatalEvent(ctx, tracer.ScopeQueue, "queue.required", "En producción, la API requiere la cola autoritativa en PostgreSQL")
		}
		queue = connection.NewJobQueue(100)
	}
	defer queue.Close()

	// Compose Layers - API does NOT run simulation or bots
	repo := repository.NewRepository(conn)
	sandbox := executor.NewSandbox(0)
	svc := service.NewService(repo, artifacts, queue, sandbox)

	srv := server.NewServer(svc)
	serverHost := fmt.Sprintf(":%s", cfg.Server.Port)
	httpServer := &http.Server{
		Addr:    serverHost,
		Handler: srv.Handler,
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		tracer.InfoEvent(ctx, tracer.ScopeHTTP, "http.stopping", "Cerrando servidor HTTP...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = httpServer.Shutdown(shutdownCtx)
		cancel()
	}()

	tracer.InfoEvent(ctx, tracer.ScopeHTTP, "http.ready", "Servidor API disponible", tracer.String("address", serverHost))
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		tracer.FatalEvent(ctx, tracer.ScopeHTTP, "http.stopped", "El servidor API se detuvo inesperadamente",
			tracer.Origin(tracer.OriginPlatform), tracer.Err(err))
	}
}
