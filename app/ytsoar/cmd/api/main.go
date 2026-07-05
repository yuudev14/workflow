package main

import (
	"context"
	"errors"
	"log"

	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/adapters/grpcserver"
	api "github.com/yuudev14/ytsoar/internal/adapters/http"
	"github.com/yuudev14/ytsoar/internal/adapters/http/handlers"
	"github.com/yuudev14/ytsoar/internal/adapters/mq"
	"github.com/yuudev14/ytsoar/internal/adapters/repository"
	"github.com/yuudev14/ytsoar/internal/adapters/ws"
	"github.com/yuudev14/ytsoar/internal/application/edges"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// @title 	YTSoar Playbook Service API
// @version	1.0
// @description YTSoar playbook service in Go using the Gin framework
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()
	appLogger := logger.SetupLogger()
	defer appLogger.Sync()

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DBUrl)
	if err != nil {
		log.Fatalf("failed to setup DB: %v", err)
	}
	defer pool.Close()
	queries := db.New(pool)
	txManager := repository.NewPgxTxManager(pool)

	hub := ws.NewHub()
	go hub.Run()

	mqConn, err := mq.Connect(cfg.MQUrl)
	if err != nil {
		log.Fatalf("failed to connect to MQ: %v", err)
	}
	defer mqConn.Close()

	// EXECUTOR picks which executor consumes triggered playbooks: the Python
	// Celery path (default queue) or the Go worker (playbook_go).
	publishQueue := cfg.PlaybookQueueName
	if cfg.Executor == "go" {
		publishQueue = cfg.GoPlaybookQueueName
	}
	taskPublisher, err := mq.NewTaskPublisher(appLogger, mqConn, publishQueue)
	if err != nil {
		log.Fatalf("failed to setup task publisher: %v", err)
	}

	statusConsumer := mq.NewStatusConsumer(appLogger, mqConn, cfg.StatusExchangeName, hub)
	go func() {
		if err := statusConsumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			appLogger.Errorf("status consumer stopped: %v", err)
		}
	}()

	playbookRepository := repository.NewPlaybookRepository(appLogger, queries, pool)
	taskRepository := repository.NewTaskRepositoryImpl(appLogger, queries, pool)
	edgeRepository := repository.NewEdgeRepositoryImpl(appLogger, queries, pool)

	playbookService := playbooks.NewPlaybookService(appLogger, playbookRepository)
	taskService := tasks.NewTaskServiceImpl(taskRepository)
	edgeService := edges.NewEdgeServiceImpl(edgeRepository)

	orchestrator := playbooks.NewPlaybookApplicationService(
		appLogger,
		playbookService,
		taskService,
		edgeService,
		txManager,
		taskPublisher,
		hub,
	)

	playbookHandler := handlers.NewPlaybookHandler(
		appLogger,
		playbookService,
		taskService,
		edgeService,
		orchestrator,
	)

	statusServer := grpcserver.NewStatusServer(appLogger, playbookService, taskService, hub)
	go func() {
		if err := statusServer.Serve(cfg.GRPCAddr); err != nil {
			log.Fatalf("grpc server: %v", err)
		}
	}()

	app := api.NewRouter(playbookHandler, hub)
	if err := app.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
