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

// fakeConnectorsTree builds a minimal connectors/ package whose core mirrors
// the real connectors/core/connector.py API (importlib loading, TOML configs)
// without the jinja2/colorlog dependencies, so the harness contract can be
// tested on any machine with python3.
func fakeConnectorsTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	core := `import importlib, inspect, tomllib


class Connector:
    @classmethod
    def get_class_container(cls, connector_id):
        module = importlib.import_module(f"connectors.{connector_id}.connector")
        for _, obj in inspect.getmembers(module):
            if inspect.isclass(obj) and issubclass(obj, Connector) and obj is not Connector:
                return obj()
        raise Exception(f"no connector class in {connector_id}")

    @classmethod
    def get_connector_config(cls, config_name, connector_id):
        if config_name:
            with open(f"./connectors/{connector_id}/configs/{config_name}.toml", "rb") as f:
                return tomllib.load(f)
        return {}

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
	mustWrite("connectors/echo_py/configs/default.toml", "key = \"value\"\n")
	return root
}

func TestPythonConnectorRunnerExecutesOperation(t *testing.T) {
	requirePython(t)
	root := fakeConnectorsTree(t)

	runner, err := localexec.NewPythonConnectorRunner(logger.NewNop(), filepath.Join(root, "connectors"), config.DefaultPythonMemoryLimitMB)
	require.NoError(t, err)

	connectorID := "echo_py"
	configName := "default"
	raw, err := runner.Execute(context.Background(), execution.ExecutionRequest{
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        "echo_node",
			ConnectorID: &connectorID,
			Config:      &configName,
			Operation:   "echo",
			Parameters:  json.RawMessage(`{"msg":"hi"}`),
		},
		PlaybookHistoryID: uuid.New(),
		Timeout:           10 * time.Second,
	})

	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "hi", decoded["echoed"])
	assert.Equal(t, "value", decoded["cfg"])
	assert.Equal(t, "echo", decoded["op"])
}

// Vendored per-connector dependencies: <id>/deps (populated by `make
// connector-deps` from <id>/requirements.txt) must be importable by the
// connector, isolated per run.
func TestPythonConnectorRunnerVendoredDeps(t *testing.T) {
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

	runner, err := localexec.NewPythonConnectorRunner(logger.NewNop(), filepath.Join(root, "connectors"), config.DefaultPythonMemoryLimitMB)
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

func TestPythonConnectorRunnerUnknownConnector(t *testing.T) {
	requirePython(t)
	root := fakeConnectorsTree(t)

	runner, err := localexec.NewPythonConnectorRunner(logger.NewNop(), filepath.Join(root, "connectors"), config.DefaultPythonMemoryLimitMB)
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

func TestPythonConnectorRunnerRequiresConnectorID(t *testing.T) {
	tree := filepath.Join(t.TempDir(), "connectors")
	require.NoError(t, os.MkdirAll(tree, 0o755))
	runner, err := localexec.NewPythonConnectorRunner(logger.NewNop(), tree, 1)
	require.NoError(t, err)

	_, err = runner.Execute(context.Background(), execution.ExecutionRequest{
		Task:    domain.Tasks{Name: "no_connector"},
		Timeout: time.Second,
	})

	assert.ErrorContains(t, err, "connector id is none")
}

func TestPythonConnectorRunnerRejectsMisnamedTree(t *testing.T) {
	_, err := localexec.NewPythonConnectorRunner(logger.NewNop(), t.TempDir(), 1)

	assert.ErrorContains(t, err, "must be a directory named 'connectors'")
}
