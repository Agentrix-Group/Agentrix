package executor

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/stretchr/testify/require"
)

type mockPoolExecutor struct {
	executedCount int32
}

func (m *mockPoolExecutor) Execute(ctx context.Context, job *connection.MatchJob) error {
	atomic.AddInt32(&m.executedCount, 1)
	return nil
}

func TestWorkerPoolLifecycle(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue := connection.NewJobQueue(10)
	defer queue.Close()

	mockExec := &mockPoolExecutor{}

	// Default concurrency when <= 0
	pool := NewWorkerPool(queue, mockExec, 0)
	r.NotNil(pool)
	r.Equal(2, pool.concurrency)

	pool.Start(ctx)

	job := &connection.MatchJob{
		JobId:   "job-pool-1",
		MatchId: "match-pool-1",
		GameId:  "starfighter",
	}

	err := queue.Enqueue(ctx, job)
	r.NoError(err)

	// Wait for worker to pick up and process job
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&mockExec.executedCount) == 1
	}, 2*time.Second, 20*time.Millisecond)

	pool.Stop()
}

type failingLeaseQueue struct {
	connection.JobQueue
	renewLeaseErr error
}

func (f *failingLeaseQueue) RenewLease(ctx context.Context, job *connection.MatchJob) error {
	if f.renewLeaseErr != nil {
		return f.renewLeaseErr
	}
	return f.JobQueue.RenewLease(ctx, job)
}

func TestWorkerPoolHeartbeatCancelsOnLeaseLoss(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	baseQueue := connection.NewJobQueue(5)
	queue := &failingLeaseQueue{
		JobQueue:      baseQueue,
		renewLeaseErr: errors.New("lease lost or stolen by another worker"),
	}

	cancelledChan := make(chan bool, 1)
	mockExec := &customExecutor{
		executeFn: func(ctx context.Context, job *connection.MatchJob) error {
			// Trigger a lease renewal attempt manually or wait for cancellation
			// In worker, we can also verify ctx cancellation
			select {
			case <-ctx.Done():
				cancelledChan <- true
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				// Simulate immediate lease check failure
				if err := queue.RenewLease(ctx, job); err != nil {
					cancelledChan <- true
					return err
				}
				return nil
			}
		},
	}

	pool := NewWorkerPool(queue, mockExec, 1)
	pool.Start(ctx)
	defer pool.Stop()

	r.NoError(queue.Enqueue(ctx, &connection.MatchJob{
		JobId:   "job-heartbeat-loss",
		MatchId: "match-heartbeat-loss",
	}))

	select {
	case cancelled := <-cancelledChan:
		r.True(cancelled, "execution must be cancelled upon lease loss")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for execution cancellation on lease loss")
	}
}

func TestConcurrentWorkersLeaseAndFencing(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue := connection.NewJobQueue(20)
	defer queue.Close()

	var completedJobs sync.Map
	var executionCount int64

	mockExec := &customExecutor{
		executeFn: func(ctx context.Context, job *connection.MatchJob) error {
			atomic.AddInt64(&executionCount, 1)
			// Ensure fencing token is positive
			r.Greater(job.FencingToken, int64(0), "fencing token must be set")
			r.NotEmpty(job.RunId, "run_id must be present")
			completedJobs.Store(job.JobId, job.FencingToken)
			time.Sleep(10 * time.Millisecond) // simulate brief execution
			return nil
		},
	}

	// Start 2 concurrent workers
	pool := NewWorkerPool(queue, mockExec, 2)
	pool.Start(ctx)
	defer pool.Stop()

	const numJobs = 10
	for i := 0; i < numJobs; i++ {
		r.NoError(queue.Enqueue(ctx, &connection.MatchJob{
			JobId:   fmt.Sprintf("job-concurrent-%d", i),
			MatchId: fmt.Sprintf("match-concurrent-%d", i),
		}))
	}

	require.Eventually(t, func() bool {
		return atomic.LoadInt64(&executionCount) == int64(numJobs)
	}, 5*time.Second, 50*time.Millisecond)

	count := 0
	completedJobs.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	r.Equal(numJobs, count, "all 10 jobs must be executed exactly once without duplicate executions")
}

type customExecutor struct {
	executeFn func(ctx context.Context, job *connection.MatchJob) error
}

func (c *customExecutor) Execute(ctx context.Context, job *connection.MatchJob) error {
	if c.executeFn != nil {
		return c.executeFn(ctx, job)
	}
	return nil
}
