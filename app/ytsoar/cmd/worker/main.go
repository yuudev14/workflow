package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/app"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func main() {
	cfg := config.Load()
	appLogger := logger.SetupLogger()
	defer func() { _ = appLogger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatalf("failed to setup DB: %v", err)
	}
	defer pool.Close()
	riverClient, err := app.NewWorker(cfg, appLogger, pool)
	if err != nil {
		log.Fatal(err)
	}
	// do not use ctx here: a SIGTERM cancels ctx and stops the running playbooks. Stop below
	// lets them finish.
	if err := riverClient.Start(context.Background()); err != nil {
		log.Fatalf("failed to start job queue: %v", err)
	}

	appLogger.Infow("worker started",
		"concurrency", cfg.PlaybookConcurrency,
		"sandbox", cfg.SandboxAddr,
	)
	<-ctx.Done()

	// Stop waits WorkerStopGrace for the running playbooks, then River cancels them. A
	// cancelled run is saved as failed, so it does not stay in_progress. The extra 30 seconds
	// let the cancelled runs save their status.
	stopCtx, cancel := context.WithTimeout(context.Background(), cfg.WorkerStopGrace+30*time.Second)
	defer cancel()
	if err := riverClient.Stop(stopCtx); err != nil {
		appLogger.Errorw("job queue did not stop cleanly", "error", err)
	}
	appLogger.Info("worker shut down")
}
