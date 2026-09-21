package integration

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/model"
	"github.com/Agentrix-Group/Agentrix/src/service"
	"github.com/stretchr/testify/require"
)

func schedule(t *testing.T, c *client, matchID string) map[string]any {
	t.Helper()
	return c.expect(http.StatusAccepted, "POST", "/api/v1/matches/"+matchID+"/runs", nil, "Idempotency-Key", "key-"+shortID()+shortID())
}

// reserveOnly takes the next job as the worker loop does.
func (e *env) reserveOnly(workerID string) *model.Reservation {
	e.t.Helper()
	res, err := e.svc.ReserveNext(context.Background(), workerID, e.engineSHA, "bubblewrap")
	require.NoError(e.t, err)
	require.NotNil(e.t, res, "no job to reserve")
	return res
}

// reserve reserves and starts the run.
func (e *env) reserve(workerID string) *model.Reservation {
	e.t.Helper()
	res := e.reserveOnly(workerID)
	require.NoError(e.t, e.svc.StartRun(context.Background(), *res))
	return res
}

// completionFor builds a well-formed completion for a reservation with a
// staged replay artifact, so tests can exercise CommitRun validation alone.
func (e *env) completionFor(res *model.Reservation) model.RunCompletion {
	e.t.Helper()
	staging, storage := service.ReplayKeys(res.RunID, "gzip")
	sha, err := e.artifacts.Put(staging, []byte("staged replay "+res.RunID))
	require.NoError(e.t, err)
	var slots []model.SlotOutcome
	for i, s := range res.Spec.Slots {
		slots = append(slots, model.SlotOutcome{SlotID: s.SlotID, Score: 10 - i, Rank: i + 1})
	}
	return model.RunCompletion{Reservation: *res, TerminationReason: "score_limit", FinalTick: 10, FinalStateHash: "h",
		Slots: slots, Replay: model.ReplayArtifact{ID: newUUID(), FormatVersion: res.Spec.ReplayFormat, Compression: "gzip",
			StagingKey: staging, StorageKey: storage, SHA256: sha, SizeBytes: int64(len("staged replay " + res.RunID)), FrameCount: 11}}
}

// P0-06: token 0, stale tokens, other workers and foreign slots are rejected.
func TestOrchestration_CommitIsFenced(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	other := e.competition(w) // second match to steal slots from
	schedule(t, comp.organizer, comp.matchID)
	schedule(t, other.organizer, other.matchID)
	res := e.reserve("w1")
	ctx := context.Background()

	good := e.completionFor(res)

	zero := good
	zero.Reservation.FencingToken = 0
	require.ErrorContains(t, e.svc.CommitRun(ctx, zero), "reservation")

	stale := good
	stale.Reservation.FencingToken = res.FencingToken - 1
	require.ErrorContains(t, e.svc.CommitRun(ctx, stale), "reservation")

	intruder := good
	intruder.Reservation.WorkerID = "w2"
	require.ErrorContains(t, e.svc.CommitRun(ctx, intruder), "reservation")

	otherMatch, err := e.store.GetMatch(ctx, other.matchID)
	require.NoError(t, err)
	foreign := good
	foreign.Slots = append([]model.SlotOutcome(nil), good.Slots...)
	foreign.Slots[1].SlotID = otherMatch.Slots[1].ID
	require.ErrorContains(t, e.svc.CommitRun(ctx, foreign), "slot")

	missing := good
	missing.Slots = good.Slots[:1]
	require.Error(t, e.svc.CommitRun(ctx, missing))

	dup := good
	dup.Slots = []model.SlotOutcome{good.Slots[0], good.Slots[0]}
	require.Error(t, e.svc.CommitRun(ctx, dup))

	badRanks := good
	badRanks.Slots = append([]model.SlotOutcome(nil), good.Slots...)
	badRanks.Slots[0].Rank, badRanks.Slots[1].Rank = 1, 3
	require.ErrorContains(t, e.svc.CommitRun(ctx, badRanks), "rank")

	foreignReplay := good
	foreignReplay.Replay.StagingKey = "replays/staging/" + newUUID() + ".ndjson.gz"
	require.ErrorContains(t, e.svc.CommitRun(ctx, foreignReplay), "replay")

	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM results`))
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM replays`))

	// The legitimate owner commits exactly once.
	require.NoError(t, e.svc.CommitRun(ctx, good))
	require.Error(t, e.svc.CommitRun(ctx, good), "second commit must be rejected")
	require.Equal(t, 2, e.count(`SELECT COUNT(*) FROM results WHERE match_run_id = $1`, res.RunID))
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_jobs WHERE run_id = $1 AND state = 'completed'`, res.RunID))
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM matches WHERE id = $1 AND state = 'finished' AND committed_run_id = $2`,
		res.MatchID, res.RunID))
}

// P0-04: a failed run is terminal; a retry is a new run with the next
// attempt and the same sealed spec.
func TestOrchestration_RetryCreatesNewRun(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	schedule(t, comp.organizer, comp.matchID)
	ctx := context.Background()

	first := e.reserve("w1")
	require.NoError(t, e.svc.FailRun(ctx, *first, model.ErrorSandbox, "sandbox spawn failed"))
	runs, err := e.store.ListRuns(ctx, comp.matchID)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	require.Equal(t, model.RunFailed, runs[0].State)
	require.Equal(t, model.RunCreated, runs[1].State)
	require.Equal(t, 2, runs[1].Attempt)
	require.Equal(t, runs[0].ExecutionSpecHash, runs[1].ExecutionSpecHash)
	match, err := e.store.GetMatch(ctx, comp.matchID)
	require.NoError(t, err)
	require.Equal(t, model.MatchQueued, match.State)

	// The failed run can never be revived.
	_, err = e.db.Exec(`UPDATE match_runs SET state = 'running' WHERE id = $1`, first.RunID)
	require.Error(t, err)
	require.Error(t, e.svc.FailRun(ctx, *first, model.ErrorSandbox, "again"))

	// Retry is delayed by backoff; advance the clock and reserve it.
	e.clock.Advance(time.Minute)
	second := e.reserve("w1")
	require.Equal(t, runs[1].ID, second.RunID)
	require.Greater(t, second.FencingToken, first.FencingToken)
	require.NoError(t, e.svc.CommitRun(ctx, e.completionFor(second)))

	// Deterministic failures are not retried.
	comp2 := e.competition(w)
	schedule(t, comp2.organizer, comp2.matchID)
	res := e.reserve("w1")
	require.NoError(t, e.svc.FailRun(ctx, *res, model.ErrorEngine, "engine crashed"))
	m2, err := e.store.GetMatch(ctx, comp2.matchID)
	require.NoError(t, err)
	require.Equal(t, model.MatchFailed, m2.State)
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_runs WHERE match_id = $1`, comp2.matchID))
	// A manual retry from failed creates attempt 2.
	ticket := schedule(t, comp2.organizer, comp2.matchID)
	require.EqualValues(t, 2, ticket["attempt"])
}

// A worker that stops heartbeating loses its lease; the reaper times the run
// out, schedules a retry and the zombie can no longer heartbeat or commit.
func TestOrchestration_LeaseExpiryRecovery(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	schedule(t, comp.organizer, comp.matchID)
	ctx := context.Background()
	zombie := e.reserve("w1")
	completion := e.completionFor(zombie)

	e.clock.Advance(2 * time.Minute)
	reaped, err := e.svc.ReapExpiredLeases(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, reaped)
	run, err := e.store.GetRun(ctx, zombie.RunID)
	require.NoError(t, err)
	require.Equal(t, model.RunTimedOut, run.State)
	require.Equal(t, model.ErrorLeaseLost, run.ErrorClass)

	require.Error(t, e.svc.Heartbeat(ctx, zombie))
	require.Error(t, e.svc.CommitRun(ctx, completion))
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM results`))

	e.clock.Advance(time.Minute)
	next := e.reserve("w1")
	require.NotEqual(t, zombie.RunID, next.RunID)
	require.Equal(t, 2, next.Attempt)
}

// Two workers polling the same queue never reserve the same job, and every
// match produces exactly one committed run.
func TestOrchestration_ConcurrentWorkersDoNotDuplicate(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	e.registerWorker("w2")
	w1 := e.worker("w1")
	w2 := e.worker("w2")
	var comps []*competition
	for i := 0; i < 3; i++ {
		c := e.competition(w1)
		schedule(t, c.organizer, c.matchID)
		comps = append(comps, c)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for _, w := range []interface{ Run(context.Context) }{w1, w2} {
		wg.Add(1)
		go func() { defer wg.Done(); w.Run(ctx) }()
	}
	for _, c := range comps {
		m := c.organizer.waitMatch(c.matchID, 90*time.Second, "finished", "failed")
		require.Equal(t, "finished", m["state"])
	}
	cancel()
	wg.Wait()
	require.Equal(t, 3, e.count(`SELECT COUNT(*) FROM match_runs WHERE state = 'committed'`))
	require.Equal(t, 3, e.count(`SELECT COUNT(*) FROM match_runs`))
	require.Equal(t, 6, e.count(`SELECT COUNT(*) FROM results`))
	// Which worker wins each job is scheduling-dependent, so the invariant is
	// checked per job: one reservation, one fencing token, all completed.
	require.Equal(t, 3, e.count(`SELECT COUNT(*) FROM match_jobs WHERE state = 'completed' AND reserved_by IN ('w1', 'w2')`))
	require.Equal(t, 3, e.count(`SELECT COUNT(DISTINCT fencing_token) FROM match_jobs`))
}

// Concurrent schedule requests: one run, identical responses for the same
// key, conflicts (never blank ids) for different keys.
func TestOrchestration_ConcurrentScheduleRun(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	comp := e.competition(e.worker("w1"))
	var wg sync.WaitGroup
	results := make([]response, 10)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := "shared-key-0001"
			if i%2 == 1 {
				key = fmt.Sprintf("other-key-%04d", i)
			}
			results[i] = comp.organizer.do("POST", "/api/v1/matches/"+comp.matchID+"/runs", nil, "Idempotency-Key", key)
		}(i)
	}
	wg.Wait()
	runIDs := map[string]bool{}
	for _, r := range results {
		switch r.Status {
		case http.StatusAccepted:
			body := r.JSON(t)
			require.NotEmpty(t, body["run_id"])
			require.NotEmpty(t, body["job_id"])
			runIDs[body["run_id"].(string)] = true
		case http.StatusConflict:
			require.Equal(t, "run_in_progress", r.ErrorCode(t))
		default:
			t.Fatalf("unexpected status %d: %s", r.Status, r.Body)
		}
	}
	require.Len(t, runIDs, 1)
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_runs`))
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_jobs`))

	// Keys are scoped per actor, and reusing one of your keys for another
	// request is a conflict, not a replay.
	other := e.competition(e.worker("w1"))
	third := e.competition(e.worker("w1"))
	res := other.organizer.do("POST", "/api/v1/matches/"+other.matchID+"/runs", nil, "Idempotency-Key", "reused-key-0001")
	require.Equal(t, http.StatusAccepted, res.Status, res.Body)
	res = third.organizer.do("POST", "/api/v1/matches/"+third.matchID+"/runs", nil, "Idempotency-Key", "reused-key-0001")
	require.Equal(t, http.StatusAccepted, res.Status, "keys are scoped per actor: %s", res.Body)
	res = other.organizer.do("POST", "/api/v1/matches/"+third.matchID+"/runs", nil, "Idempotency-Key", "reused-key-0001")
	require.Equal(t, http.StatusConflict, res.Status)
	require.Equal(t, "idempotency_key_reused", res.ErrorCode(t))
}

// Without a live worker there is no engine digest to pin: scheduling fails
// closed with 503 and nothing is queued.
func TestOrchestration_NoEngineNoRun(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	comp := e.competition(e.worker("w1"))
	e.clock.Advance(3 * time.Minute) // past the worker live window, within the token TTL
	res := comp.organizer.do("POST", "/api/v1/matches/"+comp.matchID+"/runs", nil, "Idempotency-Key", "no-engine-0001")
	require.Equal(t, http.StatusServiceUnavailable, res.Status)
	require.Equal(t, "engine_unavailable", res.ErrorCode(t))
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM match_runs`))
}

func TestOrchestration_CancelOnlyBeforeExecution(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	ticket := schedule(t, comp.organizer, comp.matchID)
	comp.organizer.expect(http.StatusOK, "POST", "/api/v1/matches/"+comp.matchID+"/cancel", map[string]string{"reason": "rescheduled"})
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_runs WHERE id = $1 AND state = 'aborted' AND error_class = 'cancelled'`, ticket["run_id"]))
	require.Equal(t, 1, e.count(`SELECT COUNT(*) FROM match_jobs WHERE state = 'cancelled'`))
	res := comp.organizer.do("POST", "/api/v1/matches/"+comp.matchID+"/runs", nil, "Idempotency-Key", "after-cancel-01")
	require.Equal(t, http.StatusConflict, res.Status)

	comp2 := e.competition(w)
	schedule(t, comp2.organizer, comp2.matchID)
	e.reserve("w1")
	res = comp2.organizer.do("POST", "/api/v1/matches/"+comp2.matchID+"/cancel", map[string]string{"reason": "late"})
	require.Equal(t, http.StatusConflict, res.Status)
}

// P0-07/P0-08: the executor rejects runs whose artifacts or engine do not
// match the sealed spec, and never replaces a competitor.
func TestOrchestration_ExecutorVerifiesDigests(t *testing.T) {
	e := newEnv(t, envOptions{})
	e.registerWorker("w1")
	w := e.worker("w1")
	comp := e.competition(w)
	schedule(t, comp.organizer, comp.matchID)
	ctx := context.Background()
	res := e.reserveOnly("w1")
	// Corrupt a submission artifact after the spec was sealed.
	require.NoError(t, e.artifacts.Delete(res.Spec.Slots[1].ArtifactKey))
	_, err := e.artifacts.Put(res.Spec.Slots[1].ArtifactKey, []byte("print('tampered')\n"))
	require.NoError(t, err)
	w.Execute(ctx, *res)
	run, err := e.store.GetRun(ctx, res.RunID)
	require.NoError(t, err)
	require.Equal(t, model.RunFailed, run.State)
	require.Equal(t, model.ErrorArtifact, run.ErrorClass)
	require.Equal(t, 0, e.count(`SELECT COUNT(*) FROM results`))
	m, err := e.store.GetMatch(ctx, comp.matchID)
	require.NoError(t, err)
	require.Equal(t, model.MatchFailed, m.State, "artifact failures are not retried")
}
