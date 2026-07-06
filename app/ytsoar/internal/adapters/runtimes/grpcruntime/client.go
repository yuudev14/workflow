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

// Client implements execution.NodeRuntime against any ConnectorRuntime gRPC
// server (the sandbox). The connection is dialed once and reused; per-call
// deadlines come from the request context.
type Client struct {
	logger logger.Logger
	client pb.ConnectorRuntimeClient
}

func New(log logger.Logger, addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		logger: log,
		client: pb.NewConnectorRuntimeClient(conn),
	}, nil
}

func (c *Client) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
	stepsJSON, err := json.Marshal(req.Steps)
	if err != nil {
		return nil, err
	}
	connectorID := ""
	if req.Task.ConnectorID != nil {
		connectorID = *req.Task.ConnectorID
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
	})
	if err != nil {
		return nil, err // transport error
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error) // connector error, includes the Python traceback
	}
	return json.RawMessage(resp.ResultJson), nil
}
