package localexec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:embed harness/python_connector_harness.py
var pythonConnectorHarness string

// PythonConnectorRunner executes Python connectors in a fresh `python3 -I`
// child, reusing connectors/core/connector.py inside the harness so importlib
// loading, TOML configs and jinja2 templating behave exactly like the Python
// service. root is the directory containing the connectors/ package.
// It implements execution.NodeRuntime.
type PythonConnectorRunner struct {
	logger logger.Logger
	root   string
}

func NewPythonConnectorRunner(log logger.Logger, root string) (*PythonConnectorRunner, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &PythonConnectorRunner{
		logger: log,
		root:   absRoot,
	}, nil
}

func (r *PythonConnectorRunner) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
	if req.Task.ConnectorID == nil {
		return nil, fmt.Errorf("connector id is none for %s", req.Task.Name)
	}
	params, err := decodeParameters(req.Task.Parameters)
	if err != nil {
		return nil, err
	}

	var configName *string
	if req.Task.Config != nil && *req.Task.Config != "" {
		configName = req.Task.Config
	}
	payload, err := json.Marshal(map[string]interface{}{
		"connectors_root": r.root,
		"connector_id":    *req.Task.ConnectorID,
		"operation":       req.Task.Operation,
		"config_name":     configName,
		"params":          params,
		"steps":           req.Steps,
	})
	if err != nil {
		return nil, err
	}

	r.logger.Debugw("running python connector",
		"connector", *req.Task.ConnectorID, "operation", req.Task.Operation)
	out, err := runSubprocess(ctx, req.Timeout, payload, scrubbedEnv(),
		"python3", "-I", "-c", pythonConnectorHarness)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
