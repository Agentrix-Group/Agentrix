package engine

import (
	"context"
	"time"
)

// StartConfig defines the operational parameters for launching an engine subprocess.
type StartConfig struct {
	BinaryPath       string
	Args             []string
	Env              []string
	WorkingDir       string
	MaxLineBytes     int           // Default: 1048576 (1 MB)
	StderrLimitBytes int           // Default: 65536 (64 KB)
	HandshakeTimeout time.Duration // Default: 5 seconds
	ShutdownTimeout  time.Duration // Default: 3 seconds
	ExpectedDigest   string        // If non-empty, binary SHA256 must match
}

// EngineClient provides a lifecycle-oriented interface for communicating with external game engines.
type EngineClient interface {
	// Start launches the engine subprocess and waits for the initial engine_ready announcement.
	Start(ctx context.Context, cfg StartConfig) error

	// InitializeMatch sends match parameters and returns initial perceptions and state hash.
	InitializeMatch(ctx context.Context, req InitializeMatchRequest) (*MatchInitializedResult, error)

	// AdvanceTick submits participant actions for the current tick and returns simulation progression.
	AdvanceTick(ctx context.Context, req AdvanceTickRequest) (*TickResult, error)

	// FinishMatch commands the engine to finalize the match early and return final rankings.
	FinishMatch(ctx context.Context, reason string) (*MatchResult, error)

	// Close requests a graceful shutdown (sending 'shutdown') and cleans up OS process resources.
	Close(ctx context.Context) error

	// EngineVersion returns the version reported by the engine during handshake.
	EngineVersion() string

	// EngineDigest returns the SHA256 hex digest of the engine executable binary.
	EngineDigest() string
}
