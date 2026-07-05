package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/adapters/mq"
	"github.com/yuudev14/ytsoar/internal/adapters/repository"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/grpcruntime"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

const (
	maxParallelNodes = 4
	nodeTimeout      = 5 * time.Minute
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

	mqConn, err := mq.Connect(cfg.MQUrl)
	if err != nil {
		log.Fatalf("failed to connect to MQ: %v", err)
	}
	defer mqConn.Close()

	statusPublisher, err := mq.NewStatusPublisher(appLogger, mqConn, cfg.StatusExchangeName)
	if err != nil {
		log.Fatalf("failed to setup status publisher: %v", err)
	}

	// The worker never runs user code: every dynamic node (python/js
	// connectors, code snippets) goes to the credential-free sandbox over
	// gRPC. Go builtin connectors later join the byConnector map here.
	sandboxRuntime, err := grpcruntime.New(appLogger, cfg.SandboxAddr)
	if err != nil {
		log.Fatalf("failed to setup sandbox runtime client: %v", err)
	}
	resolver := execution.NewStaticResolver(sandboxRuntime, nil)

	executor := execution.NewExecutor(
		appLogger,
		taskService,
		playbookService,
		resolver,
		statusPublisher,
		maxParallelNodes,
		nodeTimeout,
	)

	consumer, err := mq.NewTaskConsumer(appLogger, mqConn, cfg.GoPlaybookQueueName, executor)
	if err != nil {
		log.Fatalf("failed to setup task consumer: %v", err)
	}

	appLogger.Infow("worker started",
		"queue", cfg.GoPlaybookQueueName,
		"sandbox", cfg.SandboxAddr,
		"status_exchange", cfg.StatusExchangeName,
	)
	if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("consumer stopped: %v", err)
	}
	appLogger.Info("worker shut down")
}
