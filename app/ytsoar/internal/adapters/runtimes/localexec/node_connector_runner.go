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

//go:embed harness/node_connector_harness.ts
var nodeConnectorHarness string

// nodeConnectorEntries are the implementation files a node connector may
// ship, in resolution order — TypeScript runs via Node's native type
// stripping, so both work without a build step.
var nodeConnectorEntries = []string{"connector.ts", "connector.js"}

// NodeConnectorRunner executes JS/TS connectors from
// <tree>/<id>/connector.{ts,js} in a fresh `node` child. The connector's
// TOML config is parsed here in Go (same layout as the Python tree:
// <id>/configs/<name>.toml) and passed to the harness as JSON. It implements
// execution.NodeRuntime.
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

	payload, err := json.Marshal(map[string]any{
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
		"node", fmt.Sprintf("--max-old-space-size=%d", r.memoryLimitMB),
		"--input-type=commonjs-typescript", "-e", nodeConnectorHarness)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}

func (r *NodeConnectorRunner) loadConfig(connectorID string, configName *string) (map[string]any, error) {
	config := map[string]any{}
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

// ListNodeConnectors returns the connector ids in the unified tree whose
// info.json declares "runtime": "node" and that ship a connector.ts or
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
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name(), "info.json"))
		if err != nil {
			continue
		}
		var info struct {
			Runtime string `json:"runtime"`
		}
		if err := json.Unmarshal(raw, &info); err != nil || info.Runtime != "node" {
			continue
		}
		for _, impl := range nodeConnectorEntries {
			if _, err := os.Stat(filepath.Join(dir, entry.Name(), impl)); err == nil {
				ids = append(ids, entry.Name())
				break
			}
		}
	}
	return ids, nil
}
