package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/service"
)

type ConnectorCronWorker struct {
	cfg    config.ConnectorCronConfig
	syncer *service.ConnectorSyncService
}

func NewConnectorCronWorker(cfg config.ConnectorCronConfig, syncer *service.ConnectorSyncService) *ConnectorCronWorker {
	if cfg.OrderInterval <= 0 {
		cfg.OrderInterval = 5 * time.Minute
	}
	if cfg.PriceInterval <= 0 {
		cfg.PriceInterval = 5 * time.Minute
	}
	if cfg.MaxPages < 1 {
		cfg.MaxPages = 20
	}
	return &ConnectorCronWorker{cfg: cfg, syncer: syncer}
}

func (w *ConnectorCronWorker) Start(ctx context.Context) {
	if w == nil || !w.cfg.Enabled {
		log.Printf("connector cron disabled")
		return
	}
	if w.syncer == nil {
		log.Printf("connector cron disabled: sync service unavailable")
		return
	}
	go w.loop(ctx, "29wk_order_sync", w.cfg.OrderInterval, w.runOrderSync)
	go w.loop(ctx, "29wk_price_sync", w.cfg.PriceInterval, w.runPriceSync)
}

func (w *ConnectorCronWorker) loop(ctx context.Context, name string, interval time.Duration, run func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	log.Printf("connector cron %s scheduled every %s", name, interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			started := time.Now()
			if err := run(ctx); err != nil {
				log.Printf("connector cron %s failed after %s: %v", name, time.Since(started).Round(time.Millisecond), err)
			}
		}
	}
}

func (w *ConnectorCronWorker) runOrderSync(ctx context.Context) error {
	const jobName = "29wk_order_sync"
	if !w.syncer.JobEnabled(ctx, jobName) {
		return nil
	}
	started := w.syncer.MarkJobStarted(ctx, jobName)
	result, err := w.syncer.Sync29WKOrders(ctx, service.WK29OrderSyncInput{MaxPages: w.cfg.MaxPages})
	text := service.WK29OrderSyncSummary(result)
	if err != nil {
		text = fmt.Sprintf("%s error=%v", text, err)
	}
	w.syncer.LogSystem(ctx, "cron_29wk_order_sync", text)
	w.syncer.MarkJobFinished(ctx, jobName, started, result, err)
	return err
}

func (w *ConnectorCronWorker) runPriceSync(ctx context.Context) error {
	const jobName = "29wk_price_sync"
	if !w.syncer.JobEnabled(ctx, jobName) {
		return nil
	}
	started := w.syncer.MarkJobStarted(ctx, jobName)
	result, err := w.syncer.Sync29WKPricesAll(ctx)
	text := service.WK29PriceSyncSummary(result)
	if err != nil {
		text = fmt.Sprintf("%s error=%v", text, err)
	}
	w.syncer.LogSystem(ctx, "cron_29wk_price_sync", text)
	w.syncer.MarkJobFinished(ctx, jobName, started, result, err)
	return err
}
