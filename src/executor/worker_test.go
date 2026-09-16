package executor

import (
	"context"
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
		GameId:  "arena-basica",
	}

	err := queue.Enqueue(ctx, job)
	r.NoError(err)

	// Wait for worker to pick up and process job
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&mockExec.executedCount) == 1
	}, 2*time.Second, 20*time.Millisecond)

	pool.Stop()
}
