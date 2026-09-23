package executor

import (
	"context"
	"sync"
	"time"

	"github.com/Agentrix-Group/Agentrix/src/tracer"
)

// AdmissionProcessor ejecuta la prueba de admisión de la siguiente
// submission pendiente; devuelve false si no había trabajo.
type AdmissionProcessor interface {
	ProcessNextAdmission(ctx context.Context) (bool, error)
}

// AdmissionWorker prueba en el worker los bots subidos (ADR-0014, N4): la
// API no ejecuta bots (ADR-0007). Procesa pendientes de a uno y, sin
// trabajo, espera `idle` antes de volver a mirar.
type AdmissionWorker struct {
	processor AdmissionProcessor
	idle      time.Duration
	cancel    context.CancelFunc
	done      sync.WaitGroup
}

func NewAdmissionWorker(processor AdmissionProcessor, idle time.Duration) *AdmissionWorker {
	if idle <= 0 {
		idle = 500 * time.Millisecond
	}
	return &AdmissionWorker{processor: processor, idle: idle}
}

func (w *AdmissionWorker) Start(ctx context.Context) {
	ctx, w.cancel = context.WithCancel(ctx)
	w.done.Add(1)
	go func() {
		defer w.done.Done()
		for {
			processed, err := w.processor.ProcessNextAdmission(ctx)
			if err != nil && ctx.Err() == nil {
				tracer.WarnEvent(ctx, tracer.ScopeAgent, "submission.admission.failed", "No se pudo procesar una admisión",
					tracer.Origin(tracer.OriginInfrastructure), tracer.Err(err))
			}
			if processed && err == nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.idle):
			}
		}
	}()
}

func (w *AdmissionWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
	w.done.Wait()
}
