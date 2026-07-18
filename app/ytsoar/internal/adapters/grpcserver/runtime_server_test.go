package grpcserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	runtimepb "github.com/yuudev14/ytsoar/gen/connectorruntimepb"
	"github.com/yuudev14/ytsoar/internal/adapters/grpcserver"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	execmocks "github.com/yuudev14/ytsoar/internal/application/execution/mocks"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func TestRuntimeServerExecuteOperation(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtime := execmocks.NewMockNodeRuntime(ctrl)
	resolver := execution.NewStaticResolver(runtime, nil)
	server := grpcserver.NewRuntimeServer(logger.NewNop(), resolver, time.Minute)

	var captured execution.ExecutionRequest
	runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
			captured = req
			return json.RawMessage(`{"ok":true}`), nil
		})

	resp, err := server.ExecuteOperation(context.Background(), &runtimepb.ExecuteOperationRequest{
		ConnectorId:    "sample",
		Operation:      "op",
		ParametersJson: `{"x":1}`,
		StepsJson:      `{"A":"done"}`,
		TimeoutMs:      30000,
	})

	require.NoError(t, err)
	assert.Empty(t, resp.Error)
	assert.JSONEq(t, `{"ok":true}`, resp.ResultJson)
	assert.Equal(t, "sample", *captured.Task.ConnectorID)
	assert.Equal(t, "op", captured.Task.Operation)
	assert.Equal(t, "done", captured.Steps["A"])
	assert.Equal(t, 30*time.Second, captured.Timeout)
}

func TestRuntimeServerReturnsExecutionErrorInResponse(t *testing.T) {
	ctrl := gomock.NewController(t)
	runtime := execmocks.NewMockNodeRuntime(ctrl)
	resolver := execution.NewStaticResolver(runtime, nil)
	server := grpcserver.NewRuntimeServer(logger.NewNop(), resolver, time.Minute)

	runtime.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(nil, errors.New("boom traceback"))

	resp, err := server.ExecuteOperation(context.Background(), &runtimepb.ExecuteOperationRequest{
		ConnectorId: "sample",
		Operation:   "op",
	})

	require.NoError(t, err) // execution errors ride in the payload, not gRPC status
	assert.Contains(t, resp.Error, "boom traceback")
}

func TestRuntimeServerResolverErrorInResponse(t *testing.T) {
	resolver := execution.NewStaticResolver(nil, nil) // nothing registered
	server := grpcserver.NewRuntimeServer(logger.NewNop(), resolver, time.Minute)

	resp, err := server.ExecuteOperation(context.Background(), &runtimepb.ExecuteOperationRequest{
		ConnectorId: "unknown",
		Operation:   "op",
	})

	require.NoError(t, err)
	assert.Contains(t, resp.Error, "no runtime registered")
}

func TestRuntimeServerHealthCheck(t *testing.T) {
	server := grpcserver.NewRuntimeServer(logger.NewNop(), execution.NewStaticResolver(nil, nil), time.Minute)

	resp, err := server.HealthCheck(context.Background(), &runtimepb.HealthCheckRequest{})

	require.NoError(t, err)
	assert.True(t, resp.Ok)
}
