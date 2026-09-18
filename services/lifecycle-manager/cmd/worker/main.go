package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/lifecycle-manager/internal/worker"
)

func main() {
	once := flag.Bool("once", false, "run one bounded expiration and reconciliation pass")
	interval := flag.Duration("interval", time.Minute, "poll interval for continuous mode")
	batchSize := flag.Int("batch-size", 100, "maximum sandboxes or orphan tasks per pass")
	staleAfter := flag.Duration("stale-starting-after", 5*time.Minute, "deadline for starting sandboxes")
	operationTimeout := flag.Duration("operation-timeout", 30*time.Second, "deadline for one repository or compute operation")
	flag.Parse()

	if *interval <= 0 || *batchSize < 1 || *staleAfter <= 0 || *operationTimeout <= 0 {
		log.Fatal("interval, batch-size, stale-starting-after, and operation-timeout must be positive")
	}

	lifecycleWorker, err := configuredWorker(worker.Config{
		BatchSize:          *batchSize,
		StaleStartingAfter: *staleAfter,
		OperationTimeout:   *operationTimeout,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if *once {
		if _, err := lifecycleWorker.RunOnce(ctx, time.Now().UTC()); err != nil {
			log.Fatal(err)
		}
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for {
		if _, err := lifecycleWorker.RunOnce(ctx, time.Now().UTC()); err != nil {
			log.Printf("lifecycle pass failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// The repository and compute adapters are deployment-specific and are wired
// by the control-plane integration. Refusing to start here avoids silently
// running a fake local product mode in an AWS worker process.
func configuredWorker(_ worker.Config) (*worker.Worker, error) {
	return nil, fmt.Errorf("lifecycle worker adapters are not configured")
}
