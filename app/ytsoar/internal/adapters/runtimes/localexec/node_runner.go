package localexec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:embed harness/node_harness.ts
var nodeHarness string

// NodeRunner executes code_snippet_js nodes in a fresh `node` child with a
// capped V8 heap. The harness is TypeScript evaluated through Node's native
// type stripping (--input-type=commonjs-typescript, Node >= 23.6); the user
// snippet itself stays JavaScript. It implements execution.NodeRuntime.
type NodeRunner struct {
	logger        logger.Logger
	memoryLimitMB int
}

func NewNodeRunner(log logger.Logger) *NodeRunner {
	return &NodeRunner{
		logger:        log,
		memoryLimitMB: defaultNodeMemoryLimitMB,
	}
}

func (r *NodeRunner) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
	params, err := decodeParameters(req.Task.Parameters)
	if err != nil {
		return nil, err
	}
	if code, _ := params["code"].(string); code == "" {
		return nil, fmt.Errorf("code parameter is required for %s", req.Task.Name)
	}

	payload, err := json.Marshal(map[string]any{
		"params": params,
		"steps":  req.Steps,
	})
	if err != nil {
		return nil, err
	}

	r.logger.Debugw("running javascript code node", "task", req.Task.Name)
	out, err := runSubprocess(ctx, req.Timeout, payload, scrubbedEnv(nodePathEnv()...),
		"node", fmt.Sprintf("--max-old-space-size=%d", r.memoryLimitMB),
		"--input-type=commonjs-typescript", "-e", nodeHarness)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
