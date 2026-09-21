// Command worker executes matches. It refuses to start unless the sandbox
// can enforce isolation and limits, the engine binary answers the protocol
// handshake, and the database is the canonical baseline.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/config"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/engine"
	"github.com/Agentrix-Group/Agentrix/src/executor"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "agentrix-worker: %v\n", err)
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

	runtime, err := executor.NewRuntime(cfg.Worker.Sandbox, cfg.Worker.PodmanImage)
	if err != nil {
		return err
	}
	if err := runtime.Check(); err != nil {
		return fmt.Errorf("sandbox %s unavailable (fail closed): %w", runtime.Name(), err)
	}
	if runtime.Name() == "direct-unsandboxed" {
		tracer.WarnEvent(ctx, tracer.ScopeWorker, "sandbox.direct", "Running bots WITHOUT isolation (dev/test only)")
	}

	games, err := game.LoadRegistry(cfg.GamesDir)
	if err != nil {
		return err
	}
	modules := games.List()
	gameID := os.Getenv("WORKER_GAME")
	if gameID == "" {
		if len(modules) != 1 {
			return fmt.Errorf("WORKER_GAME is required when several game modules are installed")
		}
		gameID = modules[0].ID
	}
	module, ok := games.Get(gameID)
	if !ok {
		return fmt.Errorf("game %q is not installed", gameID)
	}
	binary := cfg.Worker.EngineBinary
	if binary == "" {
		binary = module.Engine.Binary
	}
	engineSHA, engineVersion, err := probeEngine(ctx, binary, module.EngineProtocol)
	if err != nil {
		return err
	}

	artifacts, err := connection.NewArtifactStore(cfg.ArtifactsDir)
	if err != nil {
		return err
	}
	db, err := connection.OpenDatabase(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.CheckSchemaCompatible(ctx, db); err != nil {
		return fmt.Errorf("database schema: %w", err)
	}
	// The worker never issues tokens; a throwaway secret satisfies the constructor.
	tokens, err := auth.NewTokens([]byte("worker-does-not-issue-access-tokens-0000"), time.Minute)
	if err != nil {
		return err
	}
	opts := service.DefaultOptions()
	opts.LeaseTTL, opts.MaxRunAttempts = cfg.Worker.LeaseTTL, cfg.Worker.MaxAttempts
	svc := service.New(repository.NewStore(db), games, artifacts, tokens, opts)

	if err := svc.RegisterWorker(ctx, model.WorkerInfo{ID: cfg.Worker.ID, SandboxRuntime: runtime.Name()},
		model.EngineArtifact{SHA256: engineSHA, GameID: module.ID, GameVersion: module.Version, EngineVersion: engineVersion,
			ProtocolVersion: module.EngineProtocol, SourceCommit: cfg.Worker.EngineCommit}); err != nil {
		return fmt.Errorf("register worker: %w", err)
	}
	tracer.InfoEvent(ctx, tracer.ScopeWorker, "worker.ready", "Worker registered",
		tracer.String("worker_id", cfg.Worker.ID), tracer.String("engine_sha256", engineSHA),
		tracer.String("engine_version", engineVersion), tracer.String("sandbox", runtime.Name()))

	exec := executor.NewExecutor(runtime, artifacts, executor.SubprocessEngineFactory(binary))
	worker := executor.NewWorker(svc, exec, executor.NewAdmission(runtime), executor.WorkerOptions{
		ID: cfg.Worker.ID, EngineSHA256: engineSHA, Sandbox: runtime.Name(), Concurrency: cfg.Worker.Concurrency,
		PollInterval: cfg.Worker.PollInterval, LeaseTTL: cfg.Worker.LeaseTTL, ReconcileEvery: cfg.Worker.ReconcileEvery,
		HeartbeatFile: os.Getenv("AGENTRIX_HEARTBEAT_FILE"),
	})
	worker.Run(ctx)
	tracer.InfoEvent(context.Background(), tracer.ScopeWorker, "worker.stopped", "Worker stopped")
	return nil
}

// probeEngine hashes the binary and completes a protocol handshake to learn
// the engine version the worker will announce.
func probeEngine(ctx context.Context, binary, protocol string) (string, string, error) {
	sha, _, err := connection.SHA256File(binary)
	if err != nil {
		return "", "", fmt.Errorf("engine binary %s: %w", binary, err)
	}
	client := engine.NewSubprocessClient()
	if err := client.Start(ctx, engine.StartConfig{BinaryPath: binary, ExpectedDigest: sha, HandshakeTimeout: 10 * time.Second}); err != nil {
		return "", "", fmt.Errorf("engine handshake: %w", err)
	}
	version := client.EngineVersion()
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Close(closeCtx); err != nil {
		return "", "", fmt.Errorf("engine shutdown after handshake: %w", err)
	}
	if protocol != engine.ProtocolVersion {
		return "", "", fmt.Errorf("module requires protocol %s, this worker speaks %s", protocol, engine.ProtocolVersion)
	}
	return sha, version, nil
}
