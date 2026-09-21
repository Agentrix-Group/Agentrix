package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/database"
	"github.com/Agentrix-Group/Agentrix/src/game"
	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/repository"
)

// ---------------------------------------------------------------------------
// Readiness
// ---------------------------------------------------------------------------

type HealthStatus string

const (
	Healthy  HealthStatus = "healthy"
	Degraded HealthStatus = "degraded"
	Down     HealthStatus = "down"
	Unknown  HealthStatus = "unknown"
)

type ComponentHealth struct {
	Name    string         `json:"name"`
	Status  HealthStatus   `json:"status"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type Readiness struct {
	Status     HealthStatus      `json:"status"`
	CheckedAt  time.Time         `json:"checked_at"`
	Components []ComponentHealth `json:"components"`
}

// Readiness checks every dependency and reports what it observed. A check
// that cannot run reports "unknown"; nothing is assumed healthy.
func (s *Service) Readiness(ctx context.Context) Readiness {
	now := s.now()
	r := Readiness{CheckedAt: now}
	add := func(c ComponentHealth) { r.Components = append(r.Components, c) }

	dbOK := true
	if err := s.store.Ping(ctx); err != nil {
		dbOK = false
		add(ComponentHealth{Name: "database", Status: Down, Message: "PostgreSQL is not reachable"})
	} else if err := database.CheckSchemaCompatible(ctx, s.store.DB); err != nil {
		dbOK = false
		add(ComponentHealth{Name: "database", Status: Down, Message: "schema is not the canonical baseline: " + safeSchemaMessage(err)})
	} else {
		var version string
		if err := s.store.DB.QueryRowContext(ctx, `SHOW server_version`).Scan(&version); err != nil {
			version = "unknown"
		}
		add(ComponentHealth{Name: "database", Status: Healthy, Message: "PostgreSQL reachable, schema verified",
			Details: map[string]any{"server_version": version, "schema_version": database.TargetSchemaVersion}})
	}

	if _, err := s.artifacts.Put("health/probe", []byte(now.Format(time.RFC3339Nano))); err != nil {
		add(ComponentHealth{Name: "artifacts", Status: Down, Message: "artifact store is not writable"})
	} else if err := s.artifacts.Delete("health/probe"); err != nil {
		add(ComponentHealth{Name: "artifacts", Status: Degraded, Message: "artifact store cannot delete files"})
	} else {
		add(ComponentHealth{Name: "artifacts", Status: Healthy, Message: "artifact store writable"})
	}

	modules := s.games.List()
	games := make([]map[string]any, 0, len(modules))
	for _, m := range modules {
		games = append(games, map[string]any{"id": m.ID, "version": m.Version,
			"tick_rate": fmt.Sprintf("%d/%d", m.TickRate.Numerator, m.TickRate.Denominator), "protocol": m.EngineProtocol})
	}
	add(ComponentHealth{Name: "games", Status: Healthy, Message: fmt.Sprintf("%d game module(s) loaded", len(modules)),
		Details: map[string]any{"modules": games}})

	if !dbOK {
		for _, name := range []string{"workers", "queue", "replays"} {
			add(ComponentHealth{Name: name, Status: Unknown, Message: "database unavailable"})
		}
	} else {
		workers, err := s.store.ListWorkers(ctx)
		if err != nil {
			add(ComponentHealth{Name: "workers", Status: Unknown, Message: "cannot read workers"})
		} else {
			live := []map[string]any{}
			for _, w := range workers {
				if now.Sub(w.LastSeenAt) <= s.opts.WorkerLiveWindow {
					live = append(live, map[string]any{"id": w.ID, "engine_sha256": w.EngineSHA256,
						"engine_version": w.EngineVersion, "sandbox": w.SandboxRuntime, "last_seen_at": w.LastSeenAt})
				}
			}
			status, msg := Healthy, fmt.Sprintf("%d live worker(s)", len(live))
			if len(live) == 0 {
				status, msg = Down, "no live worker: matches cannot be scheduled"
			}
			add(ComponentHealth{Name: "workers", Status: status, Message: msg, Details: map[string]any{"live": live}})
		}
		stats, err := s.store.QueueStats(ctx)
		if err != nil {
			add(ComponentHealth{Name: "queue", Status: Unknown, Message: "cannot read queue"})
		} else {
			status, msg := Healthy, "queue flowing"
			if stats.ExpiredLeases > 0 {
				status, msg = Degraded, "expired leases waiting for recovery"
			} else if stats.Pending > 0 && time.Duration(stats.OldestPendingMs)*time.Millisecond > 5*time.Minute {
				status, msg = Degraded, "pending jobs are waiting for more than 5 minutes"
			}
			add(ComponentHealth{Name: "queue", Status: status, Message: msg, Details: map[string]any{
				"pending": stats.Pending, "reserved": stats.Reserved, "expired_leases": stats.ExpiredLeases,
				"oldest_pending_ms": stats.OldestPendingMs, "fencing": "postgres sequence, token > 0 required"}})
		}
		orphans, err := s.store.OrphanCheck(ctx)
		if err != nil {
			add(ComponentHealth{Name: "integrity", Status: Unknown, Message: "cannot check run integrity"})
		} else {
			total := orphans.ActiveJobsWithoutLiveRun + orphans.RunningMatchesWithoutJob + orphans.QueuedMatchesWithoutJob +
				orphans.CommittedWithoutResults
			status, msg := Healthy, "matches, runs and jobs are consistent"
			if total > 0 {
				status, msg = Degraded, "contradictory match/run/job rows detected"
			}
			add(ComponentHealth{Name: "integrity", Status: status, Message: msg, Details: map[string]any{
				"active_jobs_without_live_run": orphans.ActiveJobsWithoutLiveRun,
				"running_matches_without_job":  orphans.RunningMatchesWithoutJob,
				"queued_matches_without_job":   orphans.QueuedMatchesWithoutJob,
				"finished_without_results":     orphans.CommittedWithoutResults}})
		}
		counts, err := s.store.ReplayCounts(ctx)
		if err != nil {
			add(ComponentHealth{Name: "replays", Status: Unknown, Message: "cannot read replays"})
		} else {
			status, msg := Healthy, "replays published"
			if counts[model.ReplayPublishFailed] > 0 {
				status, msg = Degraded, "some replays failed to publish and are being retried"
			}
			add(ComponentHealth{Name: "replays", Status: status, Message: msg, Details: map[string]any{
				"staging": counts[model.ReplayStaging], "published": counts[model.ReplayPublished],
				"publish_failed": counts[model.ReplayPublishFailed], "compression": "gzip"}})
		}
	}

	r.Status = Healthy
	for _, c := range r.Components {
		switch c.Status {
		case Down:
			r.Status = Down
		case Degraded, Unknown:
			if r.Status == Healthy {
				r.Status = Degraded
			}
		}
	}
	return r
}

func safeSchemaMessage(err error) string {
	switch {
	case errors.Is(err, database.ErrSchemaMissing):
		return "not migrated"
	case errors.Is(err, database.ErrPendingMigrations):
		return "pending migrations"
	case errors.Is(err, database.ErrSchemaDrift):
		return "schema drift detected"
	}
	return "check failed"
}

// ---------------------------------------------------------------------------
// Admission (executed by workers, never by the API)
// ---------------------------------------------------------------------------

// Admitter runs a candidate bot in the sandbox against the module's
// admission perception. A returned *AdmissionRejection is a verdict about
// the bot; any other error is an infrastructure failure (the submission
// stays validating and is retried).
type Admitter interface {
	Admit(ctx context.Context, module *game.Module, codePath string) error
}

type AdmissionRejection struct{ Reason string }

func (r *AdmissionRejection) Error() string { return r.Reason }

// AdmitNext processes one validating submission. Returns false when there
// is nothing to do.
func (s *Service) AdmitNext(ctx context.Context, admitter Admitter) (bool, error) {
	processed := false
	err := s.store.Tx(ctx, func(q *repository.Queries) error {
		id, err := q.LockNextValidatingSubmission(ctx)
		if err != nil || id == "" {
			return err
		}
		processed = true
		sub, err := q.GetSubmission(ctx, id)
		if err != nil {
			return err
		}
		agent, err := q.GetAgent(ctx, sub.AgentID)
		if err != nil {
			return err
		}
		verdict := model.SubmissionReady
		reason := ""
		module, modErr := s.module(agent.GameID)
		switch {
		case modErr != nil:
			verdict, reason = model.SubmissionRejected, "the agent's game is not installed"
		default:
			if err := s.artifacts.Verify(sub.ArtifactKey, sub.ArtifactSHA256); err != nil {
				return fmt.Errorf("admission artifact of %s: %w", sub.ID, err)
			}
			path, err := s.artifacts.Path(sub.ArtifactKey)
			if err != nil {
				return err
			}
			if err := admitter.Admit(ctx, module, path); err != nil {
				var rejection *AdmissionRejection
				if !errors.As(err, &rejection) {
					return fmt.Errorf("admission infrastructure failure: %w", err)
				}
				verdict, reason = model.SubmissionRejected, truncateText(sanitizeError(rejection.Reason), 1000)
			}
		}
		now := s.now()
		if err := q.FinishAdmission(ctx, sub.ID, verdict, reason, now); err != nil {
			return err
		}
		return q.Audit(ctx, "", "submission.admission", "submission", sub.ID, map[string]any{"status": verdict}, now)
	})
	return processed, err
}

// ---------------------------------------------------------------------------
// Reconciliation of orphaned artifacts
// ---------------------------------------------------------------------------

// CollectOrphanArtifacts removes submission blobs no row references and
// staged replays no pending row references, once older than grace.
func (s *Service) CollectOrphanArtifacts(ctx context.Context, grace time.Duration) (int, error) {
	cutoff := s.now().Add(-grace)
	removed := 0
	referenced, err := s.store.ReferencedArtifactKeys(ctx)
	if err != nil {
		return 0, err
	}
	keys, err := s.artifacts.StaleKeys("submissions/sha256", cutoff)
	if err != nil {
		return 0, err
	}
	for _, key := range keys {
		if referenced[key] {
			continue
		}
		if err := s.artifacts.RemoveRaw(key); err != nil {
			return removed, err
		}
		removed++
	}
	inUse, err := s.store.StagingKeysInUse(ctx)
	if err != nil {
		return removed, err
	}
	staged, err := s.artifacts.StaleKeys("replays/staging", cutoff)
	if err != nil {
		return removed, err
	}
	for _, key := range staged {
		if inUse[key] || strings.HasSuffix(key, ".tmp") {
			continue
		}
		if err := s.artifacts.RemoveRaw(key); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func (s *Service) PurgeExpiredIdempotency(ctx context.Context) (int64, error) {
	return s.store.PurgeExpiredIdempotency(ctx, s.now())
}

// Orphans reports contradictory rows (used by readiness tests and smoke).
func (s *Service) Orphans(ctx context.Context) (repository.OrphanCheck, error) {
	return s.store.OrphanCheck(ctx)
}
