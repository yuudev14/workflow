package localexec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/logger"
)

//go:embed harness/python_harness.py
var pythonHarness string

// PythonRunner executes code_snippet nodes in a fresh `python3 -I` child.
// It implements execution.NodeRuntime.
type PythonRunner struct {
	logger logger.Logger
}

func NewPythonRunner(log logger.Logger) *PythonRunner {
	return &PythonRunner{logger: log}
}

func (r *PythonRunner) Execute(ctx context.Context, req execution.ExecutionRequest) (json.RawMessage, error) {
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

	r.logger.Debugw("running python code node", "task", req.Task.Name)
	out, err := runSubprocess(ctx, req.Timeout, payload, scrubbedEnv(),
		"python3", "-I", "-c", pythonHarness)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(out), nil
}
