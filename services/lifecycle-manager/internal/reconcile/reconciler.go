package reconcile

import (
	"context"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/lifecycle-manager/internal/worker"
)

type Reconciler struct {
	inner *worker.Worker
}

func New(inner *worker.Worker) *Reconciler {
	return &Reconciler{inner: inner}
}

func (reconciler *Reconciler) ReconcileTasks(ctx context.Context, now time.Time, batchSize int) (int, error) {
	return reconciler.inner.ReconcileTasks(ctx, now, batchSize)
}
