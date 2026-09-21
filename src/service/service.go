// Package service implements the use cases of Agentrix. Every command that
// touches more than one row runs in a single repository transaction; every
// command validates the state machine transition it applies and the
// ownership of the resources it touches.
package service

import (
	"time"

	"github.com/Agentrix-Group/Agentrix/src/auth"
	"github.com/Agentrix-Group/Agentrix/src/connection"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
	"github.com/google/uuid"
)

type Options struct {
	RefreshIdleTTL   time.Duration
	SessionMaxTTL    time.Duration
	MaxSessions      int
	RegistrationOpen bool
	IdempotencyTTL   time.Duration
	// WorkerLiveWindow is how recent a worker heartbeat must be for its
	// engine artifact to be selected by ScheduleRun.
	WorkerLiveWindow time.Duration
	MaxRunAttempts   int
	LeaseTTL         time.Duration
}

func DefaultOptions() Options {
	return Options{
		RefreshIdleTTL: 7 * 24 * time.Hour, SessionMaxTTL: 30 * 24 * time.Hour, MaxSessions: 10,
		RegistrationOpen: true, IdempotencyTTL: 24 * time.Hour, WorkerLiveWindow: 2 * time.Minute,
		MaxRunAttempts: 3, LeaseTTL: 60 * time.Second,
	}
}

type Service struct {
	store     *repository.Store
	games     *game.Registry
	artifacts *connection.ArtifactStore
	tokens    *auth.Tokens
	opts      Options
	now       func() time.Time
	newID     func() string
}

func New(store *repository.Store, games *game.Registry, artifacts *connection.ArtifactStore, tokens *auth.Tokens, opts Options) *Service {
	return &Service{
		store: store, games: games, artifacts: artifacts, tokens: tokens, opts: opts,
		now:   func() time.Time { return time.Now().UTC().Truncate(time.Microsecond) },
		newID: uuid.NewString,
	}
}

// WithClock replaces the clock; used by tests of leases and expirations.
func (s *Service) WithClock(now func() time.Time) *Service {
	copy := *s
	copy.now = now
	return &copy
}

func (s *Service) Store() *repository.Store { return s.store }

func (s *Service) Games() *game.Registry { return s.games }

func (s *Service) Artifacts() *connection.ArtifactStore { return s.artifacts }

func (s *Service) module(gameID string) (*game.Module, error) {
	m, ok := s.games.Get(gameID)
	if !ok {
		return nil, model.Validation("game_not_supported", "game %q is not installed on this platform", gameID)
	}
	return m, nil
}

func requireCap(p model.Principal, c model.Capability) error {
	if !p.Can(c) {
		if p.Anonymous() {
			return model.Unauthorized("authentication_required", "authentication is required")
		}
		return model.Forbidden("missing_capability", "missing capability %s", c)
	}
	return nil
}
