package grpcserver

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	runtimepb "github.com/yuudev14/ytsoar/gen/connectorruntimepb"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// RuntimeServer is the sandbox's gRPC surface: it implements ConnectorRuntime
// and dispatches every request through the runtime resolver to a localexec
// subprocess runner. Execution failures travel in the response error field
// (parity with the Python runtime service), not as gRPC status errors.
type RuntimeServer struct {
	runtimepb.UnimplementedConnectorRuntimeServer
	logger         logger.Logger
	resolver       execution.RuntimeResolver
	defaultTimeout time.Duration
}

func NewRuntimeServer(log logger.Logger, resolver execution.RuntimeResolver, defaultTimeout time.Duration) *RuntimeServer {
	return &RuntimeServer{
		logger:         log,
		resolver:       resolver,
		defaultTimeout: defaultTimeout,
	}
}

func (s *RuntimeServer) ExecuteOperation(ctx context.Context, req *runtimepb.ExecuteOperationRequest) (*runtimepb.ExecuteOperationResponse, error) {
	connectorID := req.ConnectorId
	task := domain.Tasks{
		Name:        req.ConnectorId,
		ConnectorID: &connectorID,
		Operation:   req.Operation,
		Config:      req.ConfigName,
		Parameters:  json.RawMessage(req.ParametersJson),
	}
	if taskID, err := uuid.Parse(req.TaskId); err == nil {
		task.ID = taskID
	}

	steps := map[string]any{}
	if req.StepsJson != "" {
		if err := json.Unmarshal([]byte(req.StepsJson), &steps); err != nil {
			return &runtimepb.ExecuteOperationResponse{Error: "invalid steps_json: " + err.Error()}, nil
		}
	}

	timeout := s.defaultTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	historyID, _ := uuid.Parse(req.PlaybookHistoryId)

	runtime, err := s.resolver.Resolve(task)
	if err != nil {
		return &runtimepb.ExecuteOperationResponse{Error: err.Error()}, nil
	}

	result, err := runtime.Execute(ctx, execution.ExecutionRequest{
		Task:              task,
		Steps:             steps,
		PlaybookHistoryID: historyID,
		Timeout:           timeout,
	})
	if err != nil {
		s.logger.Errorw("sandbox execution failed",
			"connector", req.ConnectorId, "operation", req.Operation, "error", err)
		return &runtimepb.ExecuteOperationResponse{Error: err.Error()}, nil
	}
	return &runtimepb.ExecuteOperationResponse{ResultJson: string(result)}, nil
}

func (s *RuntimeServer) HealthCheck(ctx context.Context, req *runtimepb.HealthCheckRequest) (*runtimepb.HealthCheckResponse, error) {
	return &runtimepb.HealthCheckResponse{Ok: true}, nil
}

// Serve blocks listening on addr.
func (s *RuntimeServer) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	runtimepb.RegisterConnectorRuntimeServer(grpcServer, s)
	reflection.Register(grpcServer)
	s.logger.Infof("ConnectorRuntime gRPC server running on %v", addr)
	return grpcServer.Serve(lis)
}
