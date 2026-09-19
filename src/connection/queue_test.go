package connection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInMemoryJobQueue(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	q := NewJobQueue(5)
	r.NotNil(q)
	r.Equal(0, q.Len())

	job := &MatchJob{
		JobId:         "job-1",
		Attempt:       1,
		MatchId:       "match-1",
		ContestId:     "contest-1",
		GameId:        "starfighter",
		SubmissionIds: []string{"sub-1", "sub-2"},
		Seed:          1234,
	}

	// Enqueue
	err := q.Enqueue(ctx, job)
	r.NoError(err)
	r.Equal(1, q.Len())

	// Dequeue
	dequeued, err := q.Dequeue(ctx)
	r.NoError(err)
	r.NotNil(dequeued)
	r.Equal("job-1", dequeued.JobId)
	r.Equal(2, dequeued.Attempt)
	r.Equal(0, q.Len())
	r.NoError(q.Retry(ctx, dequeued, context.DeadlineExceeded))
	r.Equal(1, q.Len())
	retried, err := q.Dequeue(ctx)
	r.NoError(err)
	r.Equal(3, retried.Attempt)
	r.NoError(q.Complete(ctx, retried))

	// Test context cancellation when buffer is full
	fullQ := NewJobQueue(1)
	err = fullQ.Enqueue(ctx, job)
	r.NoError(err)

	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()
	err = fullQ.Enqueue(canceledCtx, job)
	r.Error(err)
	r.Equal(context.Canceled, err)

	_, err = fullQ.Dequeue(canceledCtx)
	// when channel has items, Dequeue select might pick item or context; test empty queue cancellation
	emptyQ := NewJobQueue(1)
	_, err = emptyQ.Dequeue(canceledCtx)
	r.Error(err)
	r.Equal(context.Canceled, err)

	// Close queue
	r.NoError(q.Close())

	// Enqueue on closed queue
	err = q.Enqueue(ctx, job)
	r.Error(err)

	// Dequeue on closed empty queue
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancelTimeout()
	_, err = q.Dequeue(timeoutCtx)
	r.Error(err)
}
