package localexec_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/localexec"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// fakeConnectorsTree builds a small connectors/ package whose core copies the API of the
// real connectors/core/connector.py (importlib loading) without the
// jinja2/colorlog dependencies, so the harness contract is testable on any machine with
// python3.
func fakeConnectorsTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	core := `import importlib, inspect


class Connector:
    @classmethod
    def get_class_container(cls, connector_id):
        module = importlib.import_module(f"connectors.{connector_id}.connector")
        for _, obj in inspect.getmembers(module):
            if inspect.isclass(obj) and issubclass(obj, Connector) and obj is not Connector:
                return obj()
        raise Exception(f"no connector class in {connector_id}")

    @classmethod
    def evaluate_params(cls, parameters, variables):
        return parameters or {}
`
	echo := `from connectors.core.connector import Connector


class EchoConnector(Connector):
    def execute(self, configs, params, operation, *args, **kwargs):
        return {"echoed": params.get("msg"), "cfg": configs.get("key"), "op": operation}
`
	mustWrite := func(rel, content string) {
		path := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	mustWrite("connectors/core/connector.py", core)
	mustWrite("connectors/echo_py/connector.py", echo)
	return root
}

func TestPythonConnectorRuntimeExecutesOperation(t *testing.T) {
	requirePython(t)
	root := fakeConnectorsTree(t)

	runner, err := localexec.NewPythonConnectorRuntime(logger.NewNop(), filepath.Join(root, "connectors"), config.DefaultPythonMemoryLimitMB)
	require.NoError(t, err)

	connectorID := "echo_py"
	raw, err := runner.Execute(context.Background(), execution.ExecutionRequest{
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        "echo_node",
			ConnectorID: &connectorID,
			Operation:   "echo",
			Parameters:  json.RawMessage(`{"msg":"hi"}`),
		},
		PlaybookHistoryID: uuid.New(),
		Timeout:           10 * time.Second,
		Config:            map[string]any{"key": "value"},
	})

	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "hi", decoded["echoed"])
	assert.Equal(t, "value", decoded["cfg"])
	assert.Equal(t, "echo", decoded["op"])
}

// Vendored dependencies per connector: <id>/deps (filled by `make connector-deps` from
// <id>/requirements.txt) must be importable by the connector, separate for each run.
func TestPythonConnectorRuntimeVendoredDeps(t *testing.T) {
	requirePython(t)
	root := fakeConnectorsTree(t)

	mustWrite := func(rel, content string) {
		path := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	mustWrite("connectors/dep_py/deps/fakelib.py", "VALUE = \"from-deps\"\n")
	mustWrite("connectors/dep_py/connector.py", `import fakelib

from connectors.core.connector import Connector


class DepConnector(Connector):
    def execute(self, configs, params, operation, *args, **kwargs):
        return {"dep": fakelib.VALUE}
`)

	runner, err := localexec.NewPythonConnectorRuntime(logger.NewNop(), filepath.Join(root, "connectors"), config.DefaultPythonMemoryLimitMB)
	require.NoError(t, err)

	connectorID := "dep_py"
	raw, err := runner.Execute(context.Background(), execution.ExecutionRequest{
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        "dep_node",
			ConnectorID: &connectorID,
			Operation:   "dep",
			Parameters:  json.RawMessage(`{}`),
		},
		PlaybookHistoryID: uuid.New(),
		Timeout:           10 * time.Second,
	})

	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "from-deps", decoded["dep"])
}

func TestPythonConnectorRuntimeUnknownConnector(t *testing.T) {
	requirePython(t)
	root := fakeConnectorsTree(t)

	runner, err := localexec.NewPythonConnectorRuntime(logger.NewNop(), filepath.Join(root, "connectors"), config.DefaultPythonMemoryLimitMB)
	require.NoError(t, err)

	connectorID := "ghost"
	_, err = runner.Execute(context.Background(), execution.ExecutionRequest{
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        "ghost_node",
			ConnectorID: &connectorID,
			Operation:   "noop",
			Parameters:  json.RawMessage(`{}`),
		},
		PlaybookHistoryID: uuid.New(),
		Timeout:           10 * time.Second,
	})

	assert.ErrorContains(t, err, "ghost")
}

func TestPythonConnectorRuntimeRequiresConnectorID(t *testing.T) {
	tree := filepath.Join(t.TempDir(), "connectors")
	require.NoError(t, os.MkdirAll(tree, 0o755))
	runner, err := localexec.NewPythonConnectorRuntime(logger.NewNop(), tree, 1)
	require.NoError(t, err)

	_, err = runner.Execute(context.Background(), execution.ExecutionRequest{
		Task:    domain.Tasks{Name: "no_connector"},
		Timeout: time.Second,
	})

	assert.ErrorContains(t, err, "connector id is none")
}

func TestPythonConnectorRuntimeRejectsMisnamedTree(t *testing.T) {
	_, err := localexec.NewPythonConnectorRuntime(logger.NewNop(), t.TempDir(), 1)

	assert.ErrorContains(t, err, "must be a directory named 'connectors'")
}

// The condition connector runs from the real connectors tree, so jinja2 renders the cases
// the same way it does for every other python connector.
func TestConditionConnector(t *testing.T) {
	requireJinja2(t)
	runner, err := localexec.NewPythonConnectorRuntime(logger.NewNop(), "../../../../../connectors", config.DefaultPythonMemoryLimitMB)
	require.NoError(t, err)

	run := func(t *testing.T, operation string, cases any) (map[string]any, error) {
		params, err := json.Marshal(map[string]any{"cases": cases})
		require.NoError(t, err)
		connectorID := "condition"
		raw, err := runner.Execute(context.Background(), execution.ExecutionRequest{
			Task:    domain.Tasks{ID: uuid.New(), Name: "cond", ConnectorID: &connectorID, Operation: operation, Parameters: params},
			Steps:   map[string]any{"scan": map[string]any{"score": 60, "verdict": "malicious", "d": map[string]any{"k": 1}, "n": 1}},
			Timeout: 10 * time.Second,
		})
		if err != nil {
			return nil, err
		}
		var out map[string]any
		require.NoError(t, json.Unmarshal(raw, &out))
		return out, nil
	}
	c := func(id, left, operator, right string) map[string]any {
		return map[string]any{"id": id, "left": left, "operator": operator, "right": right}
	}
	e := func(id, expression string) map[string]any {
		return map[string]any{"id": id, "expression": expression}
	}

	tests := []struct {
		name      string
		operation string
		cases     any
		expected  string
	}{
		{"switch picks the first match", "switch", []any{c("a", "x", "==", "x"), c("b", "1", "==", "1")}, "a"},
		{"switch compares plain numbers as numbers", "switch", []any{c("a", "5", "==", "5.0")}, "a"},
		{"switch compares id-like values as text", "switch", []any{c("a", "0123", "==", "123"), c("b", "0123", "!=", "123")}, "b"},
		{"switch counts a compare error as no match", "switch", []any{c("a", "abc", ">", "1"), c("b", "2", ">", "1")}, "b"},
		{"switch supports contains", "switch", []any{c("a", "hello world", "contains", "world")}, "a"},
		{"switch renders templates", "switch", []any{c("clean", `{{ var.steps["scan"].verdict }}`, "==", "clean"), c("bad", `{{ var.steps["scan"].verdict }}`, "==", "malicious")}, "bad"},
		{"switch falls to else", "switch", []any{c("a", "1", "==", "2")}, "else"},
		{"switch with no cases falls to else", "switch", nil, "else"},
		{"switch_expression picks the first truthy case", "switch_expression", []any{e("high", `{{ var.steps["scan"].score > 90 }}`), e("med", `{{ var.steps["scan"].score > 50 }}`)}, "med"},
		{"switch_expression reads empty, none and zero forms as false", "switch_expression", []any{e("a", "[]"), e("b", "None"), e("c", "0.00"), e("d", "malicious")}, "d"},
		{"switch_expression supports python methods", "switch_expression", []any{e("a", `{{ var.steps["scan"].d.get('k') == 1 }}`)}, "a"},
		{"switch_expression reads is false like jinja2", "switch_expression", []any{e("a", `{{ var.steps["scan"].n is false }}`)}, "else"},
		{"switch_expression with no cases falls to else", "switch_expression", []any{}, "else"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := run(t, tc.operation, tc.cases)
			require.NoError(t, err)
			assert.Equal(t, map[string]any{"result": tc.expected}, out)
		})
	}

	t.Run("fails when a case has no id", func(t *testing.T) {
		_, err := run(t, "switch_expression", []any{e("a", "False"), map[string]any{"expression": "True"}})
		assert.ErrorContains(t, err, "case 1 has no id")
	})
	t.Run("fails on an unknown operation", func(t *testing.T) {
		_, err := run(t, "bogus", []any{})
		assert.ErrorContains(t, err, "bogus")
	})
}
