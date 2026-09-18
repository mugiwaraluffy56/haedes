package expiration

import (
	"context"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/lifecycle-manager/internal/worker"
)

type Worker struct {
	inner *worker.Worker
}

func New(inner *worker.Worker) *Worker {
	return &Worker{inner: inner}
}

func (worker *Worker) ExpireDueSandboxes(ctx context.Context, now time.Time, batchSize int) (int, error) {
	return worker.inner.ExpireDueSandboxes(ctx, now, batchSize)
}
