package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dw0rdwk/backend/internal/cache"
	"dw0rdwk/backend/internal/config"
	"dw0rdwk/backend/internal/database"
	apphttp "dw0rdwk/backend/internal/http"
	"dw0rdwk/backend/internal/repository"
	"dw0rdwk/backend/internal/service"
	"dw0rdwk/backend/internal/worker"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	for _, warning := range cfg.Warnings() {
		log.Printf("config warning: %s", warning)
	}

	db, err := database.Open(cfg.MySQL)
	if err != nil {
		log.Fatalf("open mysql: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("open mysql pool: %v", err)
	}

	if cfg.MySQL.AutoMigrate {
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("auto migrate: %v", err)
		}
	}
	if cfg.Bootstrap.SeedDefaults {
		if err := database.SeedDefaults(db, cfg.Bootstrap); err != nil {
			log.Fatalf("seed defaults: %v", err)
		}
	}

	redisClient, err := cache.OpenRequired(context.Background(), cfg.Redis)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	repos := repository.NewRegistry(db)
	services := service.NewRegistry(repos, redisClient, cfg.Auth, cfg.Redis)
	rootCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()
	orderWorker := worker.NewOrderWorker(cfg.Worker, redisClient, repos)
	if cfg.Worker.Enabled && cfg.Worker.RecoverOnStart && redisClient.Enabled() {
		if result, err := services.Orders.RecoverStartupQueues(rootCtx, cfg.Worker.RecoveryBatchSize); err != nil {
			log.Fatalf("recover order queues: %v", err)
		} else if result.Recovered > 0 {
			log.Printf("recovered %d queued orders into redis", result.Recovered)
		}
	}
	orderWorker.Start(rootCtx)
	connectorCron := worker.NewConnectorCronWorker(cfg.ConnectorCron, services.ConnectorSync)
	connectorCron.Start(rootCtx)

	app := apphttp.NewServer(cfg, repos, services)

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- app.Listen(cfg.HTTPAddr)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-stop:
		log.Printf("shutdown requested: %s", sig)
	case err := <-listenErr:
		if err != nil {
			log.Fatalf("listen %s: %v", cfg.HTTPAddr, err)
		}
		log.Printf("server stopped")
	}

	stopWorkers()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	if redisClient.Enabled() {
		_ = redisClient.Close()
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("close mysql: %v", err)
	}
}
