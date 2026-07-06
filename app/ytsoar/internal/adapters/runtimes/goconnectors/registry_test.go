package goconnectors_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/goconnectors"
	"github.com/yuudev14/ytsoar/internal/adapters/templating"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type echoConnector struct {
	gotConfigs map[string]any
	gotParams  map[string]any
	gotOp      string
}

func (e *echoConnector) Execute(ctx context.Context, configs map[string]any, params map[string]any, operation string) (any, error) {
	e.gotConfigs, e.gotParams, e.gotOp = configs, params, operation
	return map[string]any{"echoed": params["msg"]}, nil
}

func request(t *testing.T, connectorID string, operation string, params map[string]any, steps map[string]any, config *string) execution.ExecutionRequest {
	t.Helper()
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	return execution.ExecutionRequest{
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        connectorID + "_node",
			ConnectorID: &connectorID,
			Config:      config,
			Operation:   operation,
			Parameters:  raw,
		},
		Steps:             steps,
		PlaybookHistoryID: uuid.New(),
		Timeout:           10 * time.Second,
	}
}

func TestRegistryTemplatesParamsAndLoadsConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "echo", "configs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "echo", "configs", "default.toml"),
		[]byte("key = \"value\"\n"), 0o644))

	registry := goconnectors.NewRegistry(logger.NewNop(), templating.NewGonjaEngine(), dir)
	echo := &echoConnector{}
	registry.Register("echo", echo)

	configName := "default"
	raw, err := registry.Execute(context.Background(),
		request(t, "echo", "run",
			map[string]any{"msg": `{{ var.steps["prev node"] }}`},
			map[string]any{"prev node": "hi"}, &configName))

	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "hi", decoded["echoed"])
	assert.Equal(t, "value", echo.gotConfigs["key"])
	assert.Equal(t, "run", echo.gotOp)
}

func TestRegistryUnknownConnector(t *testing.T) {
	registry := goconnectors.NewRegistry(logger.NewNop(), templating.NewGonjaEngine(), "")

	_, err := registry.Execute(context.Background(),
		request(t, "ghost", "run", nil, nil, nil))

	assert.ErrorContains(t, err, "ghost")
}

func TestRegistryIDs(t *testing.T) {
	registry := goconnectors.NewRegistry(logger.NewNop(), templating.NewGonjaEngine(), "")
	registry.Register("http_request", goconnectors.NewHTTPRequestConnector())
	registry.Register("condition", goconnectors.NewConditionConnector())

	assert.Equal(t, []string{"condition", "http_request"}, registry.IDs())
}

func TestHTTPRequestConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "yes", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items": [1, 2]}`))
	}))
	defer server.Close()

	conn := goconnectors.NewHTTPRequestConnector()
	result, err := conn.Execute(context.Background(),
		map[string]any{"headers": map[string]any{"X-Custom": "yes"}},
		map[string]any{"url": server.URL}, "get_request")

	require.NoError(t, err)
	out := result.(map[string]any)
	assert.Equal(t, http.StatusOK, out["status"])
	assert.Equal(t, map[string]any{"items": []any{float64(1), float64(2)}}, out["body"])
}

func TestHTTPRequestConnectorUnknownOperation(t *testing.T) {
	conn := goconnectors.NewHTTPRequestConnector()

	_, err := conn.Execute(context.Background(), nil, map[string]any{"url": "http://x"}, "post_request")

	assert.ErrorContains(t, err, "post_request")
}

func TestConditionConnector(t *testing.T) {
	conn := goconnectors.NewConditionConnector()
	cases := []struct {
		name     string
		params   map[string]any
		expected bool
	}{
		{"string equal", map[string]any{"left": "a", "operator": "==", "right": "a"}, true},
		{"string not equal", map[string]any{"left": "a", "operator": "!=", "right": "b"}, true},
		{"numeric equal across formats", map[string]any{"left": "5", "operator": "==", "right": "5.0"}, true},
		{"greater", map[string]any{"left": "200", "operator": ">=", "right": "200"}, true},
		{"less false", map[string]any{"left": "300", "operator": "<", "right": "200"}, false},
		{"contains", map[string]any{"left": "hello world", "operator": "contains", "right": "world"}, true},
		{"not contains", map[string]any{"left": "hello", "operator": "not_contains", "right": "x"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := conn.Execute(context.Background(), nil, tc.params, "evaluate")
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"result": tc.expected}, result)
		})
	}
}

func TestConditionConnectorErrors(t *testing.T) {
	conn := goconnectors.NewConditionConnector()

	_, err := conn.Execute(context.Background(), nil,
		map[string]any{"left": "abc", "operator": ">", "right": "1"}, "evaluate")
	assert.ErrorContains(t, err, "numeric")

	_, err = conn.Execute(context.Background(), nil,
		map[string]any{"left": "a", "operator": "~", "right": "b"}, "evaluate")
	assert.ErrorContains(t, err, "unknown operator")
}
