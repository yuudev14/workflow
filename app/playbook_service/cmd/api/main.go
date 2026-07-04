package main

import (
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
	"github.com/yuudev14/ytsoar/internal/logging"
)

// @title 	YTSoar Playbook Service API
// @version	1.0
// @description YTSoar playbook service in Go using the Gin framework
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()
	logging.Setup(cfg.LoggerMode)

	sqlxDB, err := db.SetupDB(cfg.DBUrl)
	if err != nil {
		log.Fatalf("failed to setup DB: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()

	mqConn, err := mq.Connect(cfg.MQUrl)
	if err != nil {
		log.Fatalf("failed to connect to MQ: %v", err)
	}
	defer mqConn.Close()

	taskPublisher, err := mq.NewTaskPublisher(mqConn, cfg.PlaybookQueueName)
	if err != nil {
		log.Fatalf("failed to setup task publisher: %v", err)
	}

	playbookRepository := repository.NewPlaybookRepository(sqlxDB)
	taskRepository := repository.NewTaskRepositoryImpl(sqlxDB)
	edgeRepository := repository.NewEdgeRepositoryImpl(sqlxDB)

	playbookService := playbooks.NewPlaybookService(playbookRepository)
	taskService := tasks.NewTaskServiceImpl(taskRepository)
	edgeService := edges.NewEdgeServiceImpl(edgeRepository)

	orchestrator := playbooks.NewPlaybookApplicationService(
		playbookService,
		taskService,
		edgeService,
		sqlxDB,
		taskPublisher,
		hub,
	)

	playbookHandler := handlers.NewPlaybookHandler(
		playbookService,
		taskService,
		edgeService,
		orchestrator,
	)

	statusServer := grpcserver.NewStatusServer(playbookService, taskService, hub)
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
