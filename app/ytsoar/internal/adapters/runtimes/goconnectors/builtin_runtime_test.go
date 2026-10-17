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
	registry.Register("condition", goconnectors.NewConditionConnector())

	assert.Equal(t, []string{"condition", "http_request"}, registry.IDs())
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

	_, err := conn.Execute(context.Background(), nil, map[string]any{"url": "http://x"}, "post_request")

	assert.ErrorContains(t, err, "post_request")
}

// TestConditionConnectorSwitchSimple covers the simple switch: ordered left/operator/right
// cases, and the first match wins by its stable id, else "else". A compare error (for
// example ">" on text) is no match.
func TestConditionConnectorSwitchSimple(t *testing.T) {
	conn := goconnectors.NewConditionConnector()
	cases := []struct {
		name     string
		cases    any
		expected string
	}{
		{"first matches", []any{
			map[string]any{"id": "a", "left": "x", "operator": "==", "right": "x"},
			map[string]any{"id": "b", "left": "1", "operator": "==", "right": "1"},
		}, "a"},
		{"second matches", []any{
			map[string]any{"id": "a", "left": "1", "operator": "==", "right": "2"},
			map[string]any{"id": "b", "left": "5", "operator": ">", "right": "3"},
		}, "b"},
		{"numeric across formats", []any{
			map[string]any{"id": "a", "left": "5", "operator": "==", "right": "5.0"},
		}, "a"},
		{"contains", []any{
			map[string]any{"id": "a", "left": "hello world", "operator": "contains", "right": "world"},
		}, "a"},
		{"compare error counts as no match", []any{
			map[string]any{"id": "a", "left": "abc", "operator": ">", "right": "1"},
			map[string]any{"id": "b", "left": "2", "operator": ">", "right": "1"},
		}, "b"},
		{"none match falls to else", []any{
			map[string]any{"id": "a", "left": "1", "operator": "==", "right": "2"},
		}, "else"},
		{"id-like values compare as strings", []any{
			map[string]any{"id": "a", "left": "0123", "operator": "==", "right": "123"},
			map[string]any{"id": "b", "left": "0123", "operator": "!=", "right": "123"},
		}, "b"},
		{"empty cases", []any{}, "else"},
		{"missing cases", nil, "else"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := conn.Execute(context.Background(), nil,
				map[string]any{"cases": tc.cases}, "switch")
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"result": tc.expected}, result)
		})
	}
}

func TestConditionConnectorUnknownOperation(t *testing.T) {
	conn := goconnectors.NewConditionConnector()
	_, err := conn.Execute(context.Background(), nil, map[string]any{}, "bogus")
	assert.ErrorContains(t, err, "bogus")
}

// A case without a stable id would send edges to the wrong place (position ids move when
// cases are reordered or deleted), so both operations reject it.
func TestConditionConnectorMissingCaseID(t *testing.T) {
	conn := goconnectors.NewConditionConnector()

	_, err := conn.Execute(context.Background(), nil, map[string]any{"cases": []any{
		map[string]any{"left": "1", "operator": "==", "right": "1"},
	}}, "switch")
	assert.ErrorContains(t, err, "case 0 has no id")

	_, err = conn.Execute(context.Background(), nil, map[string]any{"cases": []any{
		map[string]any{"id": "a", "expression": "False"},
		map[string]any{"expression": "True"},
	}}, "switch_expression")
	assert.ErrorContains(t, err, "case 1 has no id")
}

// TestConditionConnectorSwitchExpression covers the advanced switch: each case is a
// template expression that the registry already rendered to "True", "False" and so on. The
// id of the first truthy case wins, else "else". It also tests the truthy() conversion.
func TestConditionConnectorSwitchExpression(t *testing.T) {
	conn := goconnectors.NewConditionConnector()
	cases := []struct {
		name     string
		cases    any
		expected string
	}{
		{"first truthy", []any{
			map[string]any{"id": "a", "expression": "True"},
			map[string]any{"id": "b", "expression": "True"},
		}, "a"},
		{"second truthy", []any{
			map[string]any{"id": "a", "expression": "False"},
			map[string]any{"id": "b", "expression": "True"},
		}, "b"},
		{"none truthy falls to else", []any{
			map[string]any{"id": "a", "expression": "False"},
			map[string]any{"id": "b", "expression": ""},
		}, "else"},
		{"empty list and none are falsy, text is truthy", []any{
			map[string]any{"id": "a", "expression": "[]"},
			map[string]any{"id": "b", "expression": "None"},
			map[string]any{"id": "c", "expression": "malicious"},
		}, "c"},
		{"formatted zero is falsy", []any{
			map[string]any{"id": "a", "expression": "0.00"},
			map[string]any{"id": "b", "expression": "1"},
		}, "b"},
		{"empty cases", []any{}, "else"},
		{"missing cases", nil, "else"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := conn.Execute(context.Background(), nil,
				map[string]any{"cases": tc.cases}, "switch_expression")
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"result": tc.expected}, result)
		})
	}
}

// TestConditionSwitchExpressionThroughBuiltinRuntime tests the advanced switch end to end:
// expressions rendered by the real gonja engine pick the first matching branch.
func TestConditionSwitchExpressionThroughBuiltinRuntime(t *testing.T) {
	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine(), "")
	registry.Register("condition", goconnectors.NewConditionConnector())

	connectorID := "condition"
	steps := map[string]any{"scan": map[string]any{"score": float64(60)}}
	// score 60: first case (>90) false, second case (>50) true -> its id "med"
	params, err := json.Marshal(map[string]any{
		"cases": []map[string]any{
			{"id": "high", "expression": `{{ var.steps["scan"].score > 90 }}`},
			{"id": "med", "expression": `{{ var.steps["scan"].score > 50 }}`},
		},
	})
	require.NoError(t, err)

	raw, err := registry.Execute(context.Background(), execution.ExecutionRequest{
		Task:  domain.Tasks{Name: "cond", ConnectorID: &connectorID, Operation: "switch_expression", Parameters: params},
		Steps: steps,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"result":"med"}`, string(raw))
}

// TestConditionSwitchThroughBuiltinRuntime tests the simple switch end to end: templated
// left and right values, and the structured compare picks the first match.
func TestConditionSwitchThroughBuiltinRuntime(t *testing.T) {
	registry := goconnectors.NewBuiltinRuntime(logger.NewNop(), templating.NewGonjaEngine(), "")
	registry.Register("condition", goconnectors.NewConditionConnector())

	connectorID := "condition"
	steps := map[string]any{"scan": map[string]any{"verdict": "malicious"}}
	params, err := json.Marshal(map[string]any{
		"cases": []map[string]any{
			{"id": "clean", "left": `{{ var.steps["scan"].verdict }}`, "operator": "==", "right": "clean"},
			{"id": "bad", "left": `{{ var.steps["scan"].verdict }}`, "operator": "==", "right": "malicious"},
		},
	})
	require.NoError(t, err)

	raw, err := registry.Execute(context.Background(), execution.ExecutionRequest{
		Task:  domain.Tasks{Name: "cond", ConnectorID: &connectorID, Operation: "switch", Parameters: params},
		Steps: steps,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"result":"bad"}`, string(raw))
}
