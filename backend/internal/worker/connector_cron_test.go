package worker

import (
	"context"
	"testing"
	"time"

	"dw0rdwk/backend/internal/config"
)

func TestConnectorCronLoopStopsWithContext(t *testing.T) {
	worker := &ConnectorCronWorker{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ran := make(chan struct{}, 1)
	done := make(chan struct{})

	go func() {
		worker.loop(ctx, "test", time.Millisecond, func(context.Context) error {
			ran <- struct{}{}
			cancel()
			return nil
		})
		close(done)
	}()

	select {
	case <-ran:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("cron loop did not run")
	}
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("cron loop did not stop after context cancellation")
	}
}

func TestNewConnectorCronWorkerNormalizesDefaults(t *testing.T) {
	worker := NewConnectorCronWorker(config.ConnectorCronConfig{Enabled: true}, nil)
	if worker.cfg.OrderInterval != 5*time.Minute || worker.cfg.PriceInterval != 5*time.Minute || worker.cfg.MaxPages != 20 {
		t.Fatalf("unexpected normalized config: %+v", worker.cfg)
	}
}
