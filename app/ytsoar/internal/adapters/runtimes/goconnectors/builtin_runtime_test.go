package goconnectors_test

import (
	"context"
	"encoding/json"
	"io"
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

func TestBuiltinRuntimeTemplatesParamsAndLoadsConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "echo", "configs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "echo", "configs", "default.toml"),
		[]byte("key = \"value\"\n"), 0o644))

	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine(), dir)
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

func TestBuiltinRuntimeUnknownConnector(t *testing.T) {
	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine(), "")

	_, err := registry.Execute(context.Background(),
		request(t, "ghost", "run", nil, nil, nil))

	assert.ErrorContains(t, err, "ghost")
}

func TestBuiltinRuntimeIDs(t *testing.T) {
	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine(), "")
	registry.Register("http_request", goconnectors.NewHTTPRequestConnector())

	assert.Equal(t, []string{"http_request"}, registry.IDs())
}

func TestHTTPRequestConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "yes", r.Header.Get("X-Custom"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"items": [1, 2]}`))
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

	_, err := conn.Execute(context.Background(), nil, map[string]any{"url": "http://x"}, "patch_request")

	assert.ErrorContains(t, err, "patch_request")
}

func TestHTTPRequestConnectorMethods(t *testing.T) {
	type seen struct{ method, body, contentType, auth string }
	got := make(chan seen, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got <- seen{r.Method, string(raw), r.Header.Get("Content-Type"), r.Header.Get("Authorization")}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	conn := goconnectors.NewHTTPRequestConnector()

	cases := []struct {
		operation string
		params    map[string]any
		want      seen
	}{
		{"post_request", map[string]any{"body": `{"text":"hi"}`, "headers": `{"Authorization":"Bearer t"}`},
			seen{http.MethodPost, `{"text":"hi"}`, "application/json", "Bearer t"}},
		{"put_request", map[string]any{"body": "plain"}, seen{http.MethodPut, "plain", "", ""}},
		{"delete_request", map[string]any{}, seen{http.MethodDelete, "", "", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.operation, func(t *testing.T) {
			tc.params["url"] = server.URL
			result, err := conn.Execute(context.Background(), nil, tc.params, tc.operation)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, result.(map[string]any)["status"])
			assert.Equal(t, tc.want, <-got)
		})
	}

	t.Run("fails when the headers are not a JSON object", func(t *testing.T) {
		_, err := conn.Execute(context.Background(), nil, map[string]any{"url": server.URL, "headers": "nope"}, "post_request")
		assert.ErrorContains(t, err, "headers must be a JSON object")
	})
}
