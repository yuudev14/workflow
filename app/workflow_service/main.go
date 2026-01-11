package main

import (
	"context"
	"encoding/json"
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
	var result interface{}
	// only marshall if result is not ""
	if *req.Result != "" {
		if err := json.Unmarshal([]byte(*req.Result), &result); err != nil {
			logging.Sugar.Error("Error unmarshalling workflow params:", err)
			return nil, err
		}
		result = nil
	}
	res, err := workflowService.UpdateWorkflowHistory(req.WorkflowHistoryId, dto.UpdateWorkflowHistoryData{
		Status: types.Nullable[string]{Value: req.Status, Set: true},
		Error:  types.Nullable[string]{Value: req.Error, Set: true},
		Result: result,
	})
	// Example processing:
	return &pb.WorkflowHistory{
		Id:         res.ID.String(),
		WorkflowId: res.WorkflowID.String(),
		Status:     res.Status,
	}, err
}

func (s *server) HandleTask(ctx context.Context, req *pb.TaskStatusPayload) (*pb.TaskHistory, error) {
	workflowRepository := repository.NewWorkflowRepository(db.DB)
	taskRepository := repository.NewTaskRepositoryImpl(db.DB)
	workflowService := service.NewWorkflowService(workflowRepository)
	taskService := service.NewTaskServiceImpl(taskRepository, workflowService)
	var parameters interface{}
	// only marshall if parameters is not ""
	if *req.Parameters != "" {
		if err := json.Unmarshal([]byte(*req.Parameters), &parameters); err != nil {
			logging.Sugar.Error("Error unmarshalling parameters:", err)
			return nil, err
		}
	}
	var result interface{}
	// only marshall if result is not ""
	if req.Result != "" {
		if err := json.Unmarshal([]byte(req.Result), &result); err != nil {
			logging.Sugar.Error("Error unmarshalling result:", err)
			return nil, err
		}
	}
	res, err := taskService.UpdateTaskHistory(req.WorkflowHistoryId, req.TaskId, dto.UpdateTaskHistoryData{
		Name:          req.Name,
		Description:   req.Description,
		Parameters:    parameters,
		ConnectorName: types.Nullable[string]{Value: req.ConnectorName, Set: true},
		ConnectorID:   types.Nullable[string]{Value: req.ConnectorId, Set: true},
		Operation:     req.Operation,
		Config:        types.Nullable[string]{Value: req.Config, Set: true},
		X:             req.X,
		Y:             req.Y,
		Status:        types.Nullable[string]{Value: req.Status, Set: true},
		Error:         types.Nullable[string]{Value: req.Error, Set: true},
		Result:        result,
	})
	// Example processing:
	return &pb.TaskHistory{
		Id: res.ID.String(),
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
