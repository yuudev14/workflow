// Package localexec runs user code nodes and JS connectors inside the worker
// as fresh subprocesses. The harness scripts are embedded in the binary and
// passed inline (`python3 -I -c` / `node -e`); the payload travels over stdin
// as JSON and the result comes back on stdout as JSON, so nothing is written
// to disk. Subprocesses give fault isolation (crash/hang/OOM cannot take the
// worker down) — the worker container remains the security boundary.
package localexec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	defaultNodeMemoryLimitMB = 256

	// killGracePeriod bounds Wait after the process group is killed, so a
	// child that ignores SIGKILL semantics (unlikely) cannot hang the node.
	killGracePeriod = 2 * time.Second
)

// runSubprocess executes an interpreter in its own process group with a
// scrubbed environment, feeding payload on stdin. On timeout the whole
// process group is SIGKILLed so grandchildren die too.
func runSubprocess(
	ctx context.Context,
	timeout time.Duration,
	payload []byte,
	env []string,
	name string,
	args ...string,
) ([]byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = killGracePeriod
	cmd.Env = env
	cmd.Stdin = bytes.NewReader(payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return nil, fmt.Errorf("code execution timed out after %s", timeout)
		}
		return nil, fmt.Errorf("%s exited: %v\n%s", name, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// scrubbedEnv is the minimal environment user code may see: PATH plus the
// given extras. DB/MQ credentials from the worker process must never leak in.
func scrubbedEnv(extra ...string) []string {
	env := []string{"PATH=" + os.Getenv("PATH")}
	return append(env, extra...)
}

// nodePathEnv forwards NODE_PATH (where the harness node_modules with
// nunjucks and the npm allowlist live) when the worker has it set.
func nodePathEnv() []string {
	if nodePath := os.Getenv("NODE_PATH"); nodePath != "" {
		return []string{"NODE_PATH=" + nodePath}
	}
	return nil
}

func decodeParameters(raw json.RawMessage) (map[string]any, error) {
	params := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, fmt.Errorf("task parameters are not a JSON object: %w", err)
		}
	}
	return params, nil
}
