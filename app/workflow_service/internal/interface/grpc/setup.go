package grpc

import (
	"context"
	"encoding/json"
	"log"
	"net"

	"github.com/yuudev14-workflow/workflow-service/db"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/environment"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
	pb "github.com/yuudev14-workflow/workflow-service/internal/interface/grpc/workflows"
	"github.com/yuudev14-workflow/workflow-service/internal/tasks"
	"github.com/yuudev14-workflow/workflow-service/internal/types"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedWorkflowServer
}

func (s *server) HandleWorkflow(ctx context.Context, req *pb.WorkflowStatusPayload) (*pb.WorkflowHistory, error) {
	settings := environment.Setup()
	logging.Setup(settings.LOGGER_MODE)
	sqlxDB, err := db.SetupDB(settings.DB_URL)
	workflowRepository := workflows.NewWorkflowRepository(sqlxDB)
	workflowService := workflows.NewWorkflowService(workflowRepository)
	var result interface{}
	// only marshall if result is not None
	if req.Result != nil {
		if err := json.Unmarshal([]byte(*req.Result), &result); err != nil {
			logging.Sugar.Error("Error unmarshalling workflow params:", err)
			return nil, err
		}
		result = nil
	}
	res, err := workflowService.UpdateWorkflowHistory(req.WorkflowHistoryId, workflows.UpdateWorkflowHistoryData{
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
	settings := environment.Setup()
	logging.Setup(settings.LOGGER_MODE)
	sqlxDB, err := db.SetupDB(settings.DB_URL)
	taskRepository := tasks.NewTaskRepositoryImpl(sqlxDB)
	taskService := tasks.NewTaskServiceImpl(taskRepository)
	var parameters interface{}
	// only marshall if parameters is not None
	if req.Parameters != nil {
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
	res, err := taskService.UpdateTaskHistory(req.WorkflowHistoryId, req.TaskId, tasks.UpdateTaskHistoryData{
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

func SetupGRPCServer() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWorkflowServer(grpcServer, &server{})
	reflection.Register(grpcServer)
	log.Println("gRPC server running on :50051")
	go grpcServer.Serve(lis)

}
