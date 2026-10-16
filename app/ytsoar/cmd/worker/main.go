package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/adapters/jobs"
	"github.com/yuudev14/ytsoar/internal/adapters/pgstatus"
	"github.com/yuudev14/ytsoar/internal/adapters/repository"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/goconnectors"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/grpcruntime"
	"github.com/yuudev14/ytsoar/internal/adapters/templating"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func main() {
	cfg := config.Load()
	appLogger := logger.SetupLogger()
	defer appLogger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatalf("failed to setup DB: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	playbookRepository := repository.NewPlaybookRepository(appLogger, queries, pool)
	taskRepository := repository.NewTaskRepositoryImpl(appLogger, queries, pool)

	playbookService := playbooks.NewPlaybookService(appLogger, playbookRepository)
	taskService := tasks.NewTaskServiceImpl(taskRepository)

	// The worker never runs user code: every dynamic node (python/js/ts
	// connectors, code snippets) goes to the credential-free sandbox over
	// gRPC. Go builtins are OUR code and run in-process via the registry.
	sandboxRuntime, err := grpcruntime.New(appLogger, cfg.SandboxAddr)
	if err != nil {
		log.Fatalf("failed to setup sandbox runtime client: %v", err)
	}

	registry := goconnectors.NewRegistry(appLogger, templating.NewGonjaEngine(), cfg.ConnectorsDir)
	registry.Register("http_request", goconnectors.NewHTTPRequestConnector())
	registry.Register("condition", goconnectors.NewConditionConnector())
	byConnector := map[string]execution.NodeRuntime{}
	for _, id := range registry.IDs() {
		byConnector[id] = registry
	}
	resolver := execution.NewStaticResolver(sandboxRuntime, byConnector)

	executor := execution.NewExecutor(
		appLogger,
		taskService,
		playbookService,
		resolver,
		pgstatus.NewNotifier(pool),
		cfg.MaxParallelNodes,
		cfg.NodeTimeout,
	)

	riverClient, err := jobs.NewWorkerClient(pool,
		jobs.NewRunWorker(appLogger, playbooks.NewRunLoader(playbookService, taskService), executor),
		cfg.PlaybookConcurrency, cfg.RunTimeout)
	if err != nil {
		log.Fatalf("failed to setup job queue: %v", err)
	}
	// Do not use ctx here. A SIGTERM cancels ctx and stops the running playbooks.
	// Stop below lets them finish.
	if err := riverClient.Start(context.Background()); err != nil {
		log.Fatalf("failed to start job queue: %v", err)
	}

	appLogger.Infow("worker started",
		"concurrency", cfg.PlaybookConcurrency,
		"sandbox", cfg.SandboxAddr,
	)
	<-ctx.Done()

	// Stop waits for the running playbooks. After the run timeout, River cancels them.
	// The next worker marks a run that is still running as failed.
	stopCtx, cancel := context.WithTimeout(context.Background(), cfg.RunTimeout+time.Minute)
	defer cancel()
	if err := riverClient.Stop(stopCtx); err != nil {
		appLogger.Errorw("job queue did not stop cleanly", "error", err)
	}
	appLogger.Info("worker shut down")
}
