package grpcserver

import (
	"context"
	"encoding/json"
	"net"

	pb "github.com/yuudev14/ytsoar/gen/workflowpb"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/logging"
	"github.com/yuudev14/ytsoar/internal/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// StatusServer receives playbook/task status updates from the Python
// executor over gRPC and persists + broadcasts them.
type StatusServer struct {
	pb.UnimplementedWorkflowServer
	playbookService playbooks.PlaybookService
	taskService     tasks.TaskService
	broadcaster     contracts.StatusBroadcaster
}

func NewStatusServer(
	playbookService playbooks.PlaybookService,
	taskService tasks.TaskService,
	broadcaster contracts.StatusBroadcaster,
) *StatusServer {
	return &StatusServer{
		playbookService: playbookService,
		taskService:     taskService,
		broadcaster:     broadcaster,
	}
}

func (s *StatusServer) HandleWorkflow(ctx context.Context, req *pb.WorkflowStatusPayload) (*pb.WorkflowHistory, error) {
	var result interface{}
	// only unmarshal if result is not None
	if req.Result != nil {
		if err := json.Unmarshal([]byte(*req.Result), &result); err != nil {
			logging.Sugar.Error("Error unmarshalling playbook params:", err)
			return nil, err
		}
		result = nil
	}
	res, err := s.playbookService.UpdatePlaybookHistory(ctx, req.WorkflowHistoryId, playbooks.UpdatePlaybookHistoryData{
		Status: types.Nullable[string]{Value: req.Status, Set: true},
		Error:  types.Nullable[string]{Value: req.Error, Set: true},
		Result: result,
	})
	if err != nil {
		return nil, err
	}

	s.broadcaster.Broadcast(res)

	return &pb.WorkflowHistory{
		Id:         res.ID.String(),
		WorkflowId: res.PlaybookID.String(),
		Status:     res.Status,
	}, nil
}

func (s *StatusServer) HandleTask(ctx context.Context, req *pb.TaskStatusPayload) (*pb.TaskHistory, error) {
	var parameters interface{}
	// only unmarshal if parameters is not None
	if req.Parameters != nil {
		if err := json.Unmarshal([]byte(*req.Parameters), &parameters); err != nil {
			logging.Sugar.Error("Error unmarshalling parameters:", err)
			return nil, err
		}
	}
	var result interface{}
	// only unmarshal if result is not ""
	if req.Result != "" {
		if err := json.Unmarshal([]byte(req.Result), &result); err != nil {
			logging.Sugar.Error("Error unmarshalling result:", err)
			return nil, err
		}
	}
	res, err := s.taskService.UpdateTaskHistory(ctx, req.WorkflowHistoryId, req.TaskId, tasks.UpdateTaskHistoryData{
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
	if err != nil {
		return nil, err
	}

	return &pb.TaskHistory{
		Id: res.ID.String(),
	}, nil
}

// Serve blocks listening on addr. Run it in a goroutine from the
// composition root.
func (s *StatusServer) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	pb.RegisterWorkflowServer(grpcServer, s)
	reflection.Register(grpcServer)
	logging.Sugar.Infof("gRPC server running on %v", addr)
	return grpcServer.Serve(lis)
}
