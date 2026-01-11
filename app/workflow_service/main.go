package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/yuudev14-workflow/workflow-service/api"
	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/environment"
	pb "github.com/yuudev14-workflow/workflow-service/internal/grpc/workflows"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
	"github.com/yuudev14-workflow/workflow-service/internal/mq"
	"github.com/yuudev14-workflow/workflow-service/internal/mq/consumer"
	"github.com/yuudev14-workflow/workflow-service/internal/repository"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
	"github.com/yuudev14-workflow/workflow-service/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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

type server struct {
	pb.UnimplementedWorkflowServer
}

func (s *server) HandleWorkflow(ctx context.Context, req *pb.WorkflowStatusPayload) (*pb.WorkflowHistory, error) {
	workflowRepository := repository.NewWorkflowRepository(db.DB)
	workflowService := service.NewWorkflowService(workflowRepository)
	fmt.Println("YUUUU")
	fmt.Println(req)
	var result interface{}
	if err := json.Unmarshal([]byte(*req.Result), &result); err != nil {
		logging.Sugar.Error("Error unmarshalling workflow params:", err)
		return nil, err
	}
	res, err := workflowService.UpdateWorkflowHistory(req.WorkflowHistoryId, dto.UpdateWorkflowHistoryData{
		Status: types.Nullable[string]{Value: req.Status},
		Error:  types.Nullable[string]{Value: req.Error},
		Result: result,
	})
	// Example processing:
	return &pb.WorkflowHistory{
		Id:         res.ID.String(),
		WorkflowId: res.WorkflowID.String(),
		Status:     res.Status,
	}, err
}

func main() {

	initApp()
	mq.ConnectToMQ()
	go consumer.Listen()
	defer mq.MQConn.Close()
	defer mq.MQChannel.Close()
	app := api.InitRouter()
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWorkflowServer(grpcServer, &server{})
	reflection.Register(grpcServer)
	log.Println("gRPC server running on :50051")
	go grpcServer.Serve(lis)
	go app.Run()
	select {}
}
