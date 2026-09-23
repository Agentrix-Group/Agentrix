package executor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeAdmissionProcessor struct {
	pending atomic.Int32
	calls   atomic.Int32
	fail    atomic.Bool
}

func (p *fakeAdmissionProcessor) ProcessNextAdmission(ctx context.Context) (bool, error) {
	p.calls.Add(1)
	if p.fail.Load() {
		return false, errors.New("database unavailable")
	}
	if p.pending.Load() == 0 {
		return false, nil
	}
	p.pending.Add(-1)
	return true, nil
}

func TestAdmissionWorker_DrainsQueueWaitsWhenIdleAndStops(t *testing.T) {
	p := &fakeAdmissionProcessor{}
	p.pending.Store(3)
	w := NewAdmissionWorker(p, 20*time.Millisecond)
	w.Start(context.Background())

	deadline := time.Now().Add(2 * time.Second)
	for p.pending.Load() > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if p.pending.Load() != 0 {
		t.Fatalf("worker left %d admissions pending", p.pending.Load())
	}

	// Idle: polls every 20 ms, not in a busy loop.
	before := p.calls.Load()
	time.Sleep(200 * time.Millisecond)
	if idleCalls := p.calls.Load() - before; idleCalls > 15 {
		t.Fatalf("idle worker polled %d times in 200ms", idleCalls)
	}

	// Errors do not stop the loop; it keeps polling after the idle wait.
	p.fail.Store(true)
	time.Sleep(60 * time.Millisecond)
	p.fail.Store(false)
	p.pending.Store(1)
	deadline = time.Now().Add(2 * time.Second)
	for p.pending.Load() > 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if p.pending.Load() != 0 {
		t.Fatal("worker stopped processing after an error")
	}

	w.Stop()
	after := p.calls.Load()
	time.Sleep(60 * time.Millisecond)
	if p.calls.Load() != after {
		t.Fatal("worker kept polling after Stop")
	}
}
