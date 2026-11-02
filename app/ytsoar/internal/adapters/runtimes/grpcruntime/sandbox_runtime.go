package grpcruntime

import (
	"context"
	"encoding/json"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/yuudev14/ytsoar/gen/connectorruntimepb"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// SandboxRuntime sends a task to the sandbox over gRPC. It runs in the WORKER and fits
// execution.TaskRuntime. The sandbox picks the real runtime with its own DynamicResolver.
// The StaticResolver of the worker returns this runtime for every connector that is not
// built in. The connection is dialed once and reused. Per-call deadlines come from the
// request context.
type SandboxRuntime struct {
	logger logger.Logger
	conn   *grpc.ClientConn
	client pb.ConnectorRuntimeClient
}

func NewSandboxRuntime(log logger.Logger, addr string) (*SandboxRuntime, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &SandboxRuntime{
		logger: log,
		conn:   conn,
		client: pb.NewConnectorRuntimeClient(conn),
	}, nil
}

// Close closes the connection, call it on worker shutdown.
func (c *SandboxRuntime) Close() error {
	return c.conn.Close()
}

func (c *SandboxRuntime) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
	stepsJSON, err := json.Marshal(req.Steps)
	if err != nil {
		return nil, err
	}
	connectorID := ""
	if req.Task.ConnectorID != nil {
		connectorID = *req.Task.ConnectorID
	}

	inputJSON, err := json.Marshal(req.Input.Resolved())
	if err != nil {
		return nil, err
	}
	input := string(inputJSON)
	var config *string
	if req.Config != nil {
		raw, err := json.Marshal(req.Config)
		if err != nil {
			return nil, err
		}
		config = new(string(raw))
	}

	resp, err := c.client.ExecuteOperation(ctx, &pb.ExecuteOperationRequest{
		ConnectorId:       connectorID,
		Operation:         req.Task.Operation,
		ConfigName:        req.Task.Config,
		ParametersJson:    string(req.Task.Parameters),
		StepsJson:         string(stepsJSON),
		PlaybookHistoryId: req.PlaybookHistoryID.String(),
		TaskId:            req.Task.ID.String(),
		TimeoutMs:         uint32(req.Timeout.Milliseconds()),
		InputJson:         &input,
		ConfigJson:        config,
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return json.RawMessage(resp.ResultJson), nil
}
