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

	tracer.InfoEvent(workerCtx, tracer.ScopeWorker, "workers.ready", "Workers disponibles",
		tracer.Int("workers", p.concurrency))

	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()
			tracer.DebugEvent(workerCtx, tracer.ScopeWorker, "worker.started", "Worker iniciado", tracer.Int("worker", workerID))

			for {
				select {
				case <-workerCtx.Done():
					tracer.DebugEvent(workerCtx, tracer.ScopeWorker, "worker.stopped", "Worker detenido", tracer.Int("worker", workerID))
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
						jobCtx := tracer.WithJobID(workerCtx, job.JobId)
						jobCtx = tracer.WithMatchID(jobCtx, job.MatchId)
						jobCtx = tracer.WithAttempt(jobCtx, job.Attempt)
						tracer.DebugEvent(jobCtx, tracer.ScopeWorker, "worker.job.reserved", "Trabajo reservado", tracer.Int("worker", workerID))
						_ = p.executor.Execute(jobCtx, job)
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
