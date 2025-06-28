package main

import (
	"github.com/yuudev14-workflow/workflow-service/api"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/environment"
	"github.com/yuudev14-workflow/workflow-service/pkg/logging"
	"github.com/yuudev14-workflow/workflow-service/pkg/mq"
	"github.com/yuudev14-workflow/workflow-service/pkg/mq/consumer"
)

// @title 	Workflow Service API
// @version	1.0
// @description A Workflow Service in Go using Gin framework
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func initApp() {
	environment.Setup()
	logging.Setup(environment.Settings.LOGGER_MODE)
	db.SetupDB(environment.Settings.DB_URL)
}
func main() {
	initApp()
	mq.ConnectToMQ()
	go consumer.Listen()
	defer mq.MQConn.Close()
	defer mq.MQChannel.Close()

	app := api.InitRouter()
	go app.Run()
	select {}

}
