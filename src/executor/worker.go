package executor

import (
	"context"
	"sync"
	"time"

	"github.com/F4nk1/Agentrix/src/connection"
	"github.com/F4nk1/Agentrix/src/tracer"
)

type WorkerPool struct {
	queue       connection.JobQueue
	executor    MatchExecutor
	concurrency int
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewWorkerPool(queue connection.JobQueue, executor MatchExecutor, concurrency int) *WorkerPool {
	if concurrency <= 0 {
		concurrency = 2
	}
	return &WorkerPool{
		queue:       queue,
		executor:    executor,
		concurrency: concurrency,
	}
}

func (p *WorkerPool) Start(ctx context.Context) {
	workerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	tracer.Infof(workerCtx, "Starting worker pool with %d concurrent workers", p.concurrency)

	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			tracer.Debugf(workerCtx, "Worker %d started", workerID)

			for {
				select {
				case <-workerCtx.Done():
					tracer.Debugf(workerCtx, "Worker %d stopped", workerID)
					return
				default:
					job, err := p.queue.Dequeue(workerCtx)
					if err != nil {
						// Wait briefly before retrying or checking context
						select {
						case <-workerCtx.Done():
							return
						case <-time.After(100 * time.Millisecond):
							continue
						}
					}

					if job != nil {
						tracer.Infof(workerCtx, "Worker %d picked up match job: %s", workerID, job.MatchId)
						if err := p.executor.Execute(workerCtx, job); err != nil {
							tracer.Errorf(workerCtx, "Worker %d failed to execute match %s: %s", workerID, job.MatchId, err)
						}
					}
				}
			}
		}(i + 1)
	}
}

func (p *WorkerPool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
}
