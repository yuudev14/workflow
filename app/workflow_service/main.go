package main

import (
	"log"

	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/environment"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/mq"
	"github.com/yuudev14-workflow/workflow-service/internal/interface/api"
	"github.com/yuudev14-workflow/workflow-service/internal/interface/grpc"
)

// @title 	Workflow Service API
// @version	1.0
// @description A Workflow Service in Go using Gin framework
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {

	settings := environment.Setup()
	logging.Setup(settings.LOGGER_MODE)
	sqlxDB, err := db.SetupDB(settings.DB_URL)
	if err != nil {
		log.Fatalf("failed to setup DB: %v", err)
	}

	mqInstance := mq.ConnectToMQ(settings.MQ_URL, settings.SenderQueueName, settings.ReceiverQueueName)
	defer mqInstance.MQConn.Close()
	defer mqInstance.MQChannel.Close()

	app := api.InitRouter(sqlxDB, mqInstance)
	go grpc.SetupGRPCServer()
	go app.Run()
	select {}
}
