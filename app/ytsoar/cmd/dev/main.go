// Dev only: the api, the worker and the sandbox in one process, so one air build makes one
// binary per change. The sandbox shares the process and the DB_* settings with the api.
// Production runs the three programs apart.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

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

	// the worker calls SANDBOX_ADDR (default localhost:50052), the sandbox listens on
	// SANDBOX_LISTEN_ADDR (default :50052)
	sandbox, err := app.NewSandbox(cfg, appLogger)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Fatalf("sandbox panic: %v", r)
			}
		}()
		if err := sandbox.Serve(cfg.SandboxListenAddr); err != nil {
			log.Fatalf("sandbox server stopped: %v", err)
		}
	}()

	worker, err := app.NewWorker(cfg, appLogger, pool)
	if err != nil {
		log.Fatal(err)
	}
	if err := worker.Start(context.Background()); err != nil {
		log.Fatalf("failed to start job queue: %v", err)
	}

	router, err := app.NewAPI(ctx, cfg, appLogger, pool)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Fatalf("api panic: %v", r)
			}
		}()
		if err := router.Run(cfg.HTTPAddr); err != nil {
			log.Fatalf("http server: %v", err)
		}
	}()

	appLogger.Infow("dev started: api, worker and sandbox in one process",
		"http", cfg.HTTPAddr, "sandbox", cfg.SandboxListenAddr)
	<-ctx.Done()

	if err := worker.Stop(context.Background()); err != nil {
		appLogger.Errorw("job queue did not stop cleanly", "error", err)
	}
}
