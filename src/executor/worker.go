package executor

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

type WorkerOptions struct {
	ID             string
	EngineSHA256   string
	Sandbox        string
	Concurrency    int
	PollInterval   time.Duration
	LeaseTTL       time.Duration
	ReconcileEvery time.Duration
	OrphanGrace    time.Duration
	// HeartbeatFile, when set, is touched after every successful database
	// heartbeat (container liveness probe).
	HeartbeatFile string
}

// Worker reserves jobs for its engine digest, executes them and reports the
// outcome through the service. It also runs the reconciler: expired leases,
// submission admission, replay publication, ranking refresh and orphan GC.
type Worker struct {
	svc       *service.Service
	exec      *Executor
	admission *Admission
	opts      WorkerOptions
	wg        sync.WaitGroup
	running   atomic.Int32
}

func NewWorker(svc *service.Service, exec *Executor, admission *Admission, opts WorkerOptions) *Worker {
	if opts.OrphanGrace <= 0 {
		opts.OrphanGrace = time.Hour
	}
	return &Worker{svc: svc, exec: exec, admission: admission, opts: opts}
}

// Run blocks until ctx is cancelled and every in-flight run has finished or
// been abandoned (its lease then expires and the reaper retries it).
func (w *Worker) Run(ctx context.Context) {
	for i := 0; i < w.opts.Concurrency; i++ {
		w.wg.Add(1)
		go w.loop(ctx)
	}
	w.wg.Add(1)
	go w.reconcileLoop(ctx)
	w.wg.Wait()
}

func (w *Worker) InFlight() int { return int(w.running.Load()) }

func (w *Worker) loop(ctx context.Context) {
	defer w.wg.Done()
	for ctx.Err() == nil {
		res, err := w.svc.ReserveNext(ctx, w.opts.ID, w.opts.EngineSHA256, w.opts.Sandbox)
		if err != nil && ctx.Err() == nil {
			tracer.ErrorEvent(ctx, tracer.ScopeQueue, "worker.reserve_failed", "Job reservation failed",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
		if res == nil {
			sleep(ctx, w.opts.PollInterval)
			continue
		}
		w.running.Add(1)
		w.Execute(ctx, *res)
		w.running.Add(-1)
	}
}

// Execute runs one reservation end to end. Exported for integration tests.
func (w *Worker) Execute(ctx context.Context, res model.Reservation) {
	ctx = tracer.WithMatchID(tracer.WithRunID(tracer.WithJobID(ctx, res.JobID), res.RunID), res.MatchID)
	ctx = tracer.WithAttempt(ctx, res.Attempt)
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var lost atomic.Bool
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		ticker := time.NewTicker(w.opts.LeaseTTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-execCtx.Done():
				return
			case <-ticker.C:
				if err := w.svc.Heartbeat(context.WithoutCancel(ctx), &res); err != nil {
					lost.Store(true)
					tracer.WarnEvent(ctx, tracer.ScopeQueue, "worker.lease_lost", "Lease lost; aborting run",
						tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
					cancel()
					return
				}
			}
		}
	}()
	defer func() {
		cancel()
		<-heartbeatDone
	}()

	if err := w.svc.StartRun(ctx, res); err != nil {
		tracer.WarnEvent(ctx, tracer.ScopeQueue, "worker.start_rejected", "Run could not start",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		return
	}
	tracer.InfoEvent(ctx, tracer.ScopeMatch, "match.started", "Run started")
	completion, err := w.exec.Execute(execCtx, res)
	if lost.Load() || ctx.Err() != nil {
		w.discard(ctx, res)
		return
	}
	if err != nil {
		class := model.ErrorInfrastructure
		var runErr *RunError
		if errors.As(err, &runErr) {
			class = runErr.Class
		}
		w.discard(ctx, res)
		w.reportFailure(ctx, res, class, err)
		return
	}
	if err := w.svc.CommitRun(ctx, *completion); err != nil {
		w.discard(ctx, res)
		class := model.ErrorInfrastructure
		if kind := model.KindOf(err); kind == model.KindValidation {
			class = model.ErrorEngine
		}
		tracer.ErrorEvent(ctx, tracer.ScopeDatabase, "match.commit_failed", "Commit rejected",
			tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		w.reportFailure(ctx, res, class, err)
		return
	}
	tracer.InfoEvent(ctx, tracer.ScopeMatch, "match.completed", "Run committed",
		tracer.String("reason", completion.TerminationReason), tracer.Int("final_tick", completion.FinalTick))
	// Publication and ranking refresh are retried by the reconciler if they fail here.
	if _, err := w.svc.PublishPendingReplays(ctx, 10); err != nil {
		tracer.WarnEvent(ctx, tracer.ScopeReplay, "replay.publish_deferred", "Replay publication deferred to reconciler", tracer.Err(err))
	}
	if res.ContestID != "" {
		if err := w.svc.RecomputeRankings(ctx, res.ContestID); err != nil {
			tracer.WarnEvent(ctx, tracer.ScopeDatabase, "rankings.deferred", "Ranking refresh deferred to reconciler", tracer.Err(err))
		}
	}
}

func (w *Worker) reportFailure(ctx context.Context, res model.Reservation, class model.ErrorClass, cause error) {
	tracer.WarnEvent(ctx, tracer.ScopeMatch, "match.run_failed", "Run failed",
		tracer.String("class", string(class)), tracer.Err(cause))
	if err := w.svc.FailRun(context.WithoutCancel(ctx), res, class, cause.Error()); err != nil {
		tracer.ErrorEvent(ctx, tracer.ScopeQueue, "worker.fail_rejected",
			"Failure report rejected; the lease reaper will recover the run", tracer.Err(err))
	}
}

func (w *Worker) discard(ctx context.Context, res model.Reservation) {
	if err := w.exec.DiscardStagedReplay(res.RunID); err != nil {
		tracer.WarnEvent(ctx, tracer.ScopeReplay, "replay.discard_failed", "Staged replay not removed; GC will retry", tracer.Err(err))
	}
}

func (w *Worker) reconcileLoop(ctx context.Context) {
	defer w.wg.Done()
	lastGC := time.Time{}
	for ctx.Err() == nil {
		w.ReconcileOnce(ctx, time.Since(lastGC) > 10*time.Minute)
		if time.Since(lastGC) > 10*time.Minute {
			lastGC = time.Now()
		}
		sleep(ctx, w.opts.ReconcileEvery)
	}
}

// ReconcileOnce runs one pass of every recovery task. Each failure is
// logged and retried on the next pass; none is ignored silently.
func (w *Worker) ReconcileOnce(ctx context.Context, collectGarbage bool) {
	step := func(name string, err error) {
		if err != nil && ctx.Err() == nil {
			tracer.ErrorEvent(ctx, tracer.ScopeWorker, "reconcile."+name+"_failed", "Reconciliation step failed",
				tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
		}
	}
	err := w.svc.TouchWorker(ctx, w.opts.ID)
	step("heartbeat", err)
	if err == nil && w.opts.HeartbeatFile != "" {
		step("heartbeat_file", os.WriteFile(w.opts.HeartbeatFile, []byte(time.Now().UTC().Format(time.RFC3339)), 0o644))
	}
	_, err = w.svc.ReapExpiredLeases(ctx, 20)
	step("reap", err)
	for i := 0; i < 5 && ctx.Err() == nil; i++ {
		processed, err := w.svc.AdmitNext(ctx, w.admission)
		step("admission", err)
		if !processed || err != nil {
			break
		}
	}
	_, err = w.svc.PublishPendingReplays(ctx, 20)
	step("replays", err)
	_, err = w.svc.RecomputeDirtyRankings(ctx, 20)
	step("rankings", err)
	if collectGarbage {
		_, err = w.svc.CollectOrphanArtifacts(ctx, w.opts.OrphanGrace)
		step("orphans", err)
		_, err = w.svc.PurgeExpiredIdempotency(ctx)
		step("idempotency", err)
	}
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
