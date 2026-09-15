package connection

import (
	"context"
	"errors"
	"sync"
)

type MatchJob struct {
	JobId         string   `json:"job_id"`
	Attempt       int      `json:"attempt"`
	MatchId       string   `json:"match_id"`
	ContestId     string   `json:"contest_id"`
	GameId        string   `json:"game_id"`
	SubmissionIds []string `json:"submission_ids"`
	Seed          int64    `json:"seed"`
}

type JobQueue interface {
	Enqueue(ctx context.Context, job *MatchJob) error
	Dequeue(ctx context.Context) (*MatchJob, error)
	Close() error
	Len() int
}

type inMemoryJobQueue struct {
	mu     sync.Mutex
	ch     chan *MatchJob
	closed bool
}

func NewJobQueue(bufferSize int) JobQueue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &inMemoryJobQueue{
		ch: make(chan *MatchJob, bufferSize),
	}
}

func (q *inMemoryJobQueue) Enqueue(ctx context.Context, job *MatchJob) error {
	q.mu.Lock()
	if q.closed {
		q.mu.Unlock()
		return errors.New("queue is closed")
	}
	q.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case q.ch <- job:
		return nil
	}
}

func (q *inMemoryJobQueue) Dequeue(ctx context.Context) (*MatchJob, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case job, ok := <-q.ch:
		if !ok {
			return nil, errors.New("queue closed")
		}
		return job, nil
	}
}

func (q *inMemoryJobQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		q.closed = true
		close(q.ch)
	}
	return nil
}

func (q *inMemoryJobQueue) Len() int {
	return len(q.ch)
}
