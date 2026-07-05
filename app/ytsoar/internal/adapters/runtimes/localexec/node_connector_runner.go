package localexec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:embed harness/node_connector_harness.js
var nodeConnectorHarness string

// NodeConnectorRunner executes JS connectors from connectors-node/<id>/connector.js
// in a fresh `node` child. The connector's TOML config is parsed here in Go
// (same layout as the Python tree: <id>/configs/<name>.toml) and passed to
// the harness as JSON. It implements execution.NodeRuntime.
type NodeConnectorRunner struct {
	logger        logger.Logger
	dir           string
	memoryLimitMB int
}

func NewNodeConnectorRunner(log logger.Logger, dir string) (*NodeConnectorRunner, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &NodeConnectorRunner{
		logger:        log,
		dir:           absDir,
		memoryLimitMB: defaultNodeMemoryLimitMB,
	}, nil
}

func (r *NodeConnectorRunner) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
	if req.Task.ConnectorID == nil {
		return nil, fmt.Errorf("connector id is none for %s", req.Task.Name)
	}
	params, err := decodeParameters(req.Task.Parameters)
	if err != nil {
		return nil, err
	}
	config, err := r.loadConfig(*req.Task.ConnectorID, req.Task.Config)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(map[string]interface{}{
		"connectors_dir": r.dir,
		"connector_id":   *req.Task.ConnectorID,
		"operation":      req.Task.Operation,
		"config":         config,
		"params":         params,
		"steps":          req.Steps,
	})
	if err != nil {
		return nil, err
	}

	r.logger.Debugw("running js connector",
		"connector", *req.Task.ConnectorID, "operation", req.Task.Operation)
	out, err := runSubprocess(ctx, req.Timeout, payload, scrubbedEnv(nodePathEnv()...),
		"node", fmt.Sprintf("--max-old-space-size=%d", r.memoryLimitMB), "-e", nodeConnectorHarness)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func (r *NodeConnectorRunner) loadConfig(connectorID string, configName *string) (map[string]interface{}, error) {
	config := map[string]interface{}{}
	if configName == nil || *configName == "" {
		return config, nil
	}
	path := filepath.Join(r.dir, connectorID, "configs", *configName+".toml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read connector config %s: %w", path, err)
	}
	if err := toml.Unmarshal(raw, &config); err != nil {
		return nil, fmt.Errorf("invalid connector config %s: %w", path, err)
	}
	return config, nil
}

// ListNodeConnectors returns the connector ids under dir that ship a
// connector.js — the composition root maps each of them to this runner.
func ListNodeConnectors(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, entry.Name(), "connector.js")); err == nil {
			ids = append(ids, entry.Name())
		}
	}
	return ids, nil
}
