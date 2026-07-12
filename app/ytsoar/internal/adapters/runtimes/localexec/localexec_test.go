package localexec_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
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

const (
	testDefaultMemoryLimit = 250
)

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not installed")
	}
}

func requireJinja2(t *testing.T) {
	t.Helper()
	requirePython(t)
	if err := exec.Command("python3", "-I", "-c", "import jinja2").Run(); err != nil {
		t.Skip("jinja2 not installed")
	}
}

func requireNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed")
	}
}

// requireTypeScriptNode skips when node cannot evaluate TypeScript natively
// (--input-type=commonjs-typescript needs Node >= 23.6) — the node harnesses
// are TypeScript, so every test that executes them needs this.
func requireTypeScriptNode(t *testing.T) {
	t.Helper()
	requireNode(t)
	if err := exec.Command("node", "--input-type=commonjs-typescript", "-e", "const ok: boolean = true;").Run(); err != nil {
		t.Skip("node lacks native TypeScript type stripping (need >= 23.6)")
	}
}

func requireNunjucks(t *testing.T) {
	t.Helper()
	requireTypeScriptNode(t)
	if err := exec.Command("node", "-e", `require("nunjucks")`).Run(); err != nil {
		t.Skip("nunjucks not resolvable via NODE_PATH")
	}
}

func codeRequest(t *testing.T, code string, extraParams map[string]any, steps map[string]any, timeout time.Duration) execution.ExecutionRequest {
	t.Helper()
	params := map[string]any{"code": code}
	for key, value := range extraParams {
		params[key] = value
	}
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	connectorID := "code_snippet_py"
	return execution.ExecutionRequest{
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        "code_node",
			ConnectorID: &connectorID,
			Operation:   "code",
			Parameters:  raw,
		},
		Steps:             steps,
		PlaybookHistoryID: uuid.New(),
		Timeout:           timeout,
	}
}

func codeOutput(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	return decoded["code_output"]
}

func TestPythonRunnerReturnsResult(t *testing.T) {
	requirePython(t)
	runner := localexec.NewPythonRunner(logger.NewNop(), config.DefaultPythonMemoryLimitMB)

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "result = params['x'] + len(steps)", map[string]any{"x": 41},
			map[string]any{"A": "done"}, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, float64(42), codeOutput(t, raw))
}

func TestPythonRunnerTemplating(t *testing.T) {
	requireJinja2(t)
	runner := localexec.NewPythonRunner(logger.NewNop(), config.DefaultPythonMemoryLimitMB)

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, `result = params["greeting"]`,
			map[string]any{"greeting": `{{ var.steps["A"] }}`},
			map[string]any{"A": "hello"}, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, "hello", codeOutput(t, raw))
}

func TestPythonRunnerErrorIncludesTraceback(t *testing.T) {
	requirePython(t)
	runner := localexec.NewPythonRunner(logger.NewNop(), config.DefaultPythonMemoryLimitMB)

	_, err := runner.Execute(context.Background(),
		codeRequest(t, "raise Exception('boom')", nil, nil, 10*time.Second))

	assert.ErrorContains(t, err, "boom")
}

func TestPythonRunnerTimeoutKillsProcess(t *testing.T) {
	requirePython(t)
	runner := localexec.NewPythonRunner(logger.NewNop(), config.DefaultPythonMemoryLimitMB)

	start := time.Now()
	_, err := runner.Execute(context.Background(),
		codeRequest(t, "while True: pass", nil, nil, time.Second))

	assert.ErrorContains(t, err, "timed out")
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestPythonRunnerEnvIsScrubbed(t *testing.T) {
	requirePython(t)
	t.Setenv("DB_PASSWORD", "supersecret")
	runner := localexec.NewPythonRunner(logger.NewNop(), config.DefaultPythonMemoryLimitMB)

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "import os\nresult = os.environ.get('DB_PASSWORD', 'scrubbed')",
			nil, nil, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, "scrubbed", codeOutput(t, raw))
}

func TestPythonRunnerRequiresCode(t *testing.T) {
	runner := localexec.NewPythonRunner(logger.NewNop(), config.DefaultPythonMemoryLimitMB)
	req := codeRequest(t, "", nil, nil, time.Second)

	_, err := runner.Execute(context.Background(), req)

	assert.ErrorContains(t, err, "code parameter is required")
}

func TestNodeRunnerReturnsResult(t *testing.T) {
	requireTypeScriptNode(t)
	runner := localexec.NewNodeRunner(logger.NewNop(), config.DefaultNodeMemoryLimitMB)

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "const result = await Promise.resolve(params.x * 2);",
			map[string]any{"x": 21}, nil, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, float64(42), codeOutput(t, raw))
}

func TestNodeRunnerTemplating(t *testing.T) {
	requireNunjucks(t)
	runner := localexec.NewNodeRunner(logger.NewNop(), config.DefaultNodeMemoryLimitMB)

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "const result = params.greeting;",
			map[string]any{"greeting": `{{ var.steps["A"] }}`},
			map[string]any{"A": "hello"}, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, "hello", codeOutput(t, raw))
}

func TestNodeRunnerErrorIsCaptured(t *testing.T) {
	requireTypeScriptNode(t)
	runner := localexec.NewNodeRunner(logger.NewNop(), config.DefaultNodeMemoryLimitMB)

	_, err := runner.Execute(context.Background(),
		codeRequest(t, "throw new Error('boom');", nil, nil, 10*time.Second))

	assert.ErrorContains(t, err, "boom")
}

func TestNodeRunnerTimeoutKillsProcess(t *testing.T) {
	requireTypeScriptNode(t)
	runner := localexec.NewNodeRunner(logger.NewNop(), config.DefaultNodeMemoryLimitMB)

	start := time.Now()
	_, err := runner.Execute(context.Background(),
		codeRequest(t, "while (true) {}", nil, nil, time.Second))

	assert.ErrorContains(t, err, "timed out")
	assert.Less(t, time.Since(start), 5*time.Second)
}

// realTSCore is the actual core shipped in the repo tree, relative to this
// package directory — the tests exercise the real class-discovery/templating
// contract, not a fake. The implementation is TypeScript; plain-JS connectors
// reach it through the 1-line connector.js shim (extensionless require never
// resolves .ts), so the temp tree replicates both files.
const realTSCore = "../../../../../connectors/core/connector.ts"

const jsCoreShim = `module.exports = require("./connector.ts");`

func writeCore(t *testing.T, dir string) {
	t.Helper()
	core, err := os.ReadFile(realTSCore)
	require.NoError(t, err, "repo TS core must exist at %s", realTSCore)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "core", "connector.ts"), core, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "core", "connector.js"), []byte(jsCoreShim), 0o644))
}

func writeNodeConnector(t *testing.T, dir string) {
	t.Helper()
	writeCore(t, dir)

	connectorDir := filepath.Join(dir, "echo_js")
	require.NoError(t, os.MkdirAll(filepath.Join(connectorDir, "configs"), 0o755))
	connectorJS := `const { Connector } = require("../core/connector");

class EchoConnector extends Connector {
  async execute(configs, params, operation) {
    const operations = {
      echo: () => ({ echoed: params.msg, cfg: configs.key, op: operation }),
    };
    const handler = operations[operation];
    if (!handler) {
      throw new Error("operation (" + operation + ") does not exist in EchoConnector");
    }
    return handler();
  }

  async healthCheck() {
    return true;
  }
}

module.exports = { EchoConnector };`
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "connector.js"), []byte(connectorJS), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "info.json"),
		[]byte(`{"id":"echo_js","name":"Echo JS","runtime":"node"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "configs", "default.toml"), []byte("key = \"value\"\n"), 0o644))
}

// writeTSConnector adds a TypeScript connector (connector.ts, no .js) to the
// tree — Node executes it directly via native type stripping.
func writeTSConnector(t *testing.T, dir string) {
	t.Helper()

	connectorDir := filepath.Join(dir, "echo_ts")
	require.NoError(t, os.MkdirAll(filepath.Join(connectorDir, "configs"), 0o755))
	connectorTS := `const { Connector } = require("../core/connector.ts");

type Values = Record<string, unknown>;

class EchoTS extends Connector {
  async execute(configs: Values, params: Values, operation: string): Promise<unknown> {
    if (operation !== "echo") {
      throw new Error("operation (" + operation + ") does not exist in EchoTS");
    }
    return { echoed: params.msg, cfg: configs.key, op: operation };
  }

  async healthCheck(): Promise<boolean> {
    return true;
  }
}

module.exports = { EchoTS };`
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "connector.ts"), []byte(connectorTS), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "info.json"),
		[]byte(`{"id":"echo_ts","name":"Echo TS","runtime":"node"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "configs", "default.toml"), []byte("key = \"value\"\n"), 0o644))
}

func TestNodeConnectorRunnerExecutesOperation(t *testing.T) {
	requireTypeScriptNode(t)
	dir := t.TempDir()
	writeNodeConnector(t, dir)

	runner, err := localexec.NewNodeConnectorRunner(logger.NewNop(), dir, testDefaultMemoryLimit)
	require.NoError(t, err)

	connectorID := "echo_js"
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

func TestNodeConnectorRunnerExecutesTypeScriptConnector(t *testing.T) {
	requireTypeScriptNode(t)
	dir := t.TempDir()
	writeCore(t, dir)
	writeTSConnector(t, dir)

	runner, err := localexec.NewNodeConnectorRunner(logger.NewNop(), dir, testDefaultMemoryLimit)
	require.NoError(t, err)

	connectorID := "echo_ts"
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

// Vendored per-connector dependencies: <id>/node_modules (populated by `make
// connector-deps` from <id>/package.json) resolve through Node's standard
// upward walk from the connector file — no harness involvement.
func TestNodeConnectorRunnerVendoredDeps(t *testing.T) {
	requireTypeScriptNode(t)
	dir := t.TempDir()
	writeCore(t, dir)

	connectorDir := filepath.Join(dir, "dep_js")
	libDir := filepath.Join(connectorDir, "node_modules", "fakelib")
	require.NoError(t, os.MkdirAll(libDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "package.json"),
		[]byte(`{"name":"fakelib","version":"1.0.0","main":"index.js"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "index.js"),
		[]byte(`module.exports = { value: "from-node-modules" };`), 0o644))
	connectorJS := `const { Connector } = require("../core/connector");
const fakelib = require("fakelib");

class DepConnector extends Connector {
  async execute(configs, params, operation) {
    return { dep: fakelib.value };
  }
}

module.exports = { DepConnector };`
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "connector.js"), []byte(connectorJS), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "info.json"),
		[]byte(`{"id":"dep_js","name":"Dep JS","runtime":"node"}`), 0o644))

	runner, err := localexec.NewNodeConnectorRunner(logger.NewNop(), dir, testDefaultMemoryLimit)
	require.NoError(t, err)

	connectorID := "dep_js"
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
	assert.Equal(t, "from-node-modules", decoded["dep"])
}

func TestNodeConnectorRunnerUnknownOperation(t *testing.T) {
	requireTypeScriptNode(t)
	dir := t.TempDir()
	writeNodeConnector(t, dir)

	runner, err := localexec.NewNodeConnectorRunner(logger.NewNop(), dir, testDefaultMemoryLimit)
	require.NoError(t, err)

	connectorID := "echo_js"
	_, err = runner.Execute(context.Background(), execution.ExecutionRequest{
		Task: domain.Tasks{
			ID:          uuid.New(),
			Name:        "echo_node",
			ConnectorID: &connectorID,
			Operation:   "missing_op",
			Parameters:  json.RawMessage(`{}`),
		},
		PlaybookHistoryID: uuid.New(),
		Timeout:           10 * time.Second,
	})

	assert.ErrorContains(t, err, "missing_op")
}

func TestListNodeConnectors(t *testing.T) {
	dir := t.TempDir()
	writeNodeConnector(t, dir)
	writeTSConnector(t, dir) // ships only connector.ts — still listed
	// python connectors, dirs without info.json and plain files are ignored
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "python_connector"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "python_connector", "info.json"),
		[]byte(`{"id":"python_connector"}`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "not_a_connector"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("x"), 0o644))

	ids, err := localexec.ListNodeConnectors(dir)

	require.NoError(t, err)
	assert.Equal(t, []string{"echo_js", "echo_ts"}, ids)
}
