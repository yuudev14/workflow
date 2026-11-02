package localexec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:embed harness/node_connector_harness.ts
var nodeConnectorHarness string

// nodeConnectorEntries: the files a node connector can ship, in lookup order. TypeScript
// runs with Node's type stripping, so both work with no build step.
var nodeConnectorEntries = []string{"connector.ts", "connector.js"}

// NodeJSConnectorRuntime runs a JS or TS connector from <tree>/<id>/connector.{ts,js} in a
// new `node` process. It runs in the SANDBOX and fits execution.TaskRuntime. The opened
// config goes to the harness as JSON.
type NodeJSConnectorRuntime struct {
	logger        logger.Logger
	dir           string
	memoryLimitMB int
}

func NewNodeJSConnectorRuntime(log logger.Logger, dir string, memoryLimitMB int) (*NodeJSConnectorRuntime, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if memoryLimitMB <= 0 {
		memoryLimitMB = defaultNodeMemoryLimitMB
	}
	return &NodeJSConnectorRuntime{
		logger:        log,
		dir:           absDir,
		memoryLimitMB: memoryLimitMB,
	}, nil
}

func (r *NodeJSConnectorRuntime) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
	if req.Task.ConnectorID == nil {
		return nil, fmt.Errorf("connector id is none for %s", req.Task.Name)
	}
	params, err := decodeParameters(req.Task.Parameters)
	if err != nil {
		return nil, err
	}
	config := req.Config
	if config == nil {
		config = map[string]any{}
	}

	payload, err := json.Marshal(map[string]any{
		"connectors_dir": r.dir,
		"connector_id":   *req.Task.ConnectorID,
		"operation":      req.Task.Operation,
		"config":         config,
		"params":         params,
		"steps":          req.Steps,
		"input":          req.Input.TemplateVars(),
	})
	if err != nil {
		return nil, err
	}

	r.logger.Debugw("running js connector",
		"connector", *req.Task.ConnectorID, "operation", req.Task.Operation)
	out, err := runSubprocess(ctx, req.Timeout, payload, scrubbedEnv(nodePathEnv()...),
		"node", fmt.Sprintf("--max-old-space-size=%d", r.memoryLimitMB),
		"--input-type=commonjs-typescript", "-e", nodeConnectorHarness)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

// IsNodeJSConnector says if info.json has "runtime": "node" and the connector ships a
// connector.ts or connector.js. It is checked per request, so a connector uploaded after
// boot works without a restart.
func IsNodeJSConnector(dir, id string) bool {
	raw, err := os.ReadFile(filepath.Join(dir, id, "info.json"))
	if err != nil {
		return false
	}
	var info struct {
		Runtime string `json:"runtime"`
	}
	if err := json.Unmarshal(raw, &info); err != nil || info.Runtime != "node" {
		return false
	}
	for _, impl := range nodeConnectorEntries {
		if _, err := os.Stat(filepath.Join(dir, id, impl)); err == nil {
			return true
		}
	}
	return false
}

// ListNodeConnectors returns the connector ids in the tree that IsNodeJSConnector accepts.
func ListNodeConnectors(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() && IsNodeJSConnector(dir, entry.Name()) {
			ids = append(ids, entry.Name())
		}
	}
	return ids, nil
}
