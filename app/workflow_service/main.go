package main

import (
	"context"
	"log"

	"github.com/yuudev14-workflow/workflow-service/api"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/environment"
	pb "github.com/yuudev14-workflow/workflow-service/internal/buffers"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/mq"
	"github.com/yuudev14-workflow/workflow-service/internal/mq/consumer"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWorkflowServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(_ context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

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
	s := grpc.NewServer()
	pb.RegisterWorkflowServer()
	go app.Run()
	select {}
}
