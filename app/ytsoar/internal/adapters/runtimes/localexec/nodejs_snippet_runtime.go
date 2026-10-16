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

// NodeJSSnippetRuntime runs a code_snippet_js task in a fresh `node` process with a capped
// V8 heap. It runs in the SANDBOX and fits execution.TaskRuntime. The harness is TypeScript,
// which Node runs with native type stripping (--input-type=commonjs-typescript, Node >= 23.6).
// The user snippet itself stays JavaScript.
type NodeJSSnippetRuntime struct {
	logger        logger.Logger
	memoryLimitMB int
}

func NewNodeJSSnippetRuntime(log logger.Logger, memoryLimitMB int) *NodeJSSnippetRuntime {
	if memoryLimitMB <= 0 {
		memoryLimitMB = defaultNodeMemoryLimitMB
	}
	return &NodeJSSnippetRuntime{
		logger:        log,
		memoryLimitMB: memoryLimitMB,
	}
}

func (r *NodeJSSnippetRuntime) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
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
		"input":  req.Input.TemplateVars(),
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
