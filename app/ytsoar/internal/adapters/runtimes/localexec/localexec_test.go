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
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
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

func requireNunjucks(t *testing.T) {
	t.Helper()
	requireNode(t)
	if err := exec.Command("node", "-e", `require("nunjucks")`).Run(); err != nil {
		t.Skip("nunjucks not resolvable via NODE_PATH")
	}
}

func codeRequest(t *testing.T, code string, extraParams map[string]interface{}, steps map[string]interface{}, timeout time.Duration) execution.ExecutionRequest {
	t.Helper()
	params := map[string]interface{}{"code": code}
	for key, value := range extraParams {
		params[key] = value
	}
	raw, err := json.Marshal(params)
	require.NoError(t, err)
	connectorID := "code_snippet"
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

func codeOutput(t *testing.T, raw json.RawMessage) interface{} {
	t.Helper()
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	return decoded["code_output"]
}

func TestPythonRunnerReturnsResult(t *testing.T) {
	requirePython(t)
	runner := localexec.NewPythonRunner(logger.NewNop())

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "result = params['x'] + len(steps)", map[string]interface{}{"x": 41},
			map[string]interface{}{"A": "done"}, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, float64(42), codeOutput(t, raw))
}

func TestPythonRunnerTemplating(t *testing.T) {
	requireJinja2(t)
	runner := localexec.NewPythonRunner(logger.NewNop())

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, `result = params["greeting"]`,
			map[string]interface{}{"greeting": `{{ var.steps["A"] }}`},
			map[string]interface{}{"A": "hello"}, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, "hello", codeOutput(t, raw))
}

func TestPythonRunnerErrorIncludesTraceback(t *testing.T) {
	requirePython(t)
	runner := localexec.NewPythonRunner(logger.NewNop())

	_, err := runner.Execute(context.Background(),
		codeRequest(t, "raise Exception('boom')", nil, nil, 10*time.Second))

	assert.ErrorContains(t, err, "boom")
}

func TestPythonRunnerTimeoutKillsProcess(t *testing.T) {
	requirePython(t)
	runner := localexec.NewPythonRunner(logger.NewNop())

	start := time.Now()
	_, err := runner.Execute(context.Background(),
		codeRequest(t, "while True: pass", nil, nil, time.Second))

	assert.ErrorContains(t, err, "timed out")
	assert.Less(t, time.Since(start), 5*time.Second)
}

func TestPythonRunnerEnvIsScrubbed(t *testing.T) {
	requirePython(t)
	t.Setenv("DB_PASSWORD", "supersecret")
	runner := localexec.NewPythonRunner(logger.NewNop())

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "import os\nresult = os.environ.get('DB_PASSWORD', 'scrubbed')",
			nil, nil, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, "scrubbed", codeOutput(t, raw))
}

func TestPythonRunnerRequiresCode(t *testing.T) {
	runner := localexec.NewPythonRunner(logger.NewNop())
	req := codeRequest(t, "", nil, nil, time.Second)

	_, err := runner.Execute(context.Background(), req)

	assert.ErrorContains(t, err, "code parameter is required")
}

func TestNodeRunnerReturnsResult(t *testing.T) {
	requireNode(t)
	runner := localexec.NewNodeRunner(logger.NewNop())

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "const result = await Promise.resolve(params.x * 2);",
			map[string]interface{}{"x": 21}, nil, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, float64(42), codeOutput(t, raw))
}

func TestNodeRunnerTemplating(t *testing.T) {
	requireNunjucks(t)
	runner := localexec.NewNodeRunner(logger.NewNop())

	raw, err := runner.Execute(context.Background(),
		codeRequest(t, "const result = params.greeting;",
			map[string]interface{}{"greeting": `{{ var.steps["A"] }}`},
			map[string]interface{}{"A": "hello"}, 10*time.Second))

	require.NoError(t, err)
	assert.Equal(t, "hello", codeOutput(t, raw))
}

func TestNodeRunnerErrorIsCaptured(t *testing.T) {
	requireNode(t)
	runner := localexec.NewNodeRunner(logger.NewNop())

	_, err := runner.Execute(context.Background(),
		codeRequest(t, "throw new Error('boom');", nil, nil, 10*time.Second))

	assert.ErrorContains(t, err, "boom")
}

func TestNodeRunnerTimeoutKillsProcess(t *testing.T) {
	requireNode(t)
	runner := localexec.NewNodeRunner(logger.NewNop())

	start := time.Now()
	_, err := runner.Execute(context.Background(),
		codeRequest(t, "while (true) {}", nil, nil, time.Second))

	assert.ErrorContains(t, err, "timed out")
	assert.Less(t, time.Since(start), 5*time.Second)
}

func writeNodeConnector(t *testing.T, dir string) {
	t.Helper()
	connectorDir := filepath.Join(dir, "echo_js")
	require.NoError(t, os.MkdirAll(filepath.Join(connectorDir, "configs"), 0o755))
	connectorJS := `module.exports.operations = {
  async echo(config, params) {
    return { echoed: params.msg, cfg: config.key };
  },
};`
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "connector.js"), []byte(connectorJS), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(connectorDir, "configs", "default.toml"), []byte("key = \"value\"\n"), 0o644))
}

func TestNodeConnectorRunnerExecutesOperation(t *testing.T) {
	requireNode(t)
	dir := t.TempDir()
	writeNodeConnector(t, dir)

	runner, err := localexec.NewNodeConnectorRunner(logger.NewNop(), dir)
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
	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, "hi", decoded["echoed"])
	assert.Equal(t, "value", decoded["cfg"])
}

func TestNodeConnectorRunnerUnknownOperation(t *testing.T) {
	requireNode(t)
	dir := t.TempDir()
	writeNodeConnector(t, dir)

	runner, err := localexec.NewNodeConnectorRunner(logger.NewNop(), dir)
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
	// a dir without connector.js and a plain file must both be ignored
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "not_a_connector"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("x"), 0o644))

	ids, err := localexec.ListNodeConnectors(dir)

	require.NoError(t, err)
	assert.Equal(t, []string{"echo_js"}, ids)
}
