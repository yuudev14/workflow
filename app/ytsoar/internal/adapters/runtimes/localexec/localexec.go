// Package localexec runs user code nodes and JS connectors in the worker as new
// subprocesses. The harness scripts are embedded in the binary and passed inline
// (`python3 -I -c` / `node -e`). The payload goes in over stdin as JSON and the result
// comes back on stdout as JSON, so nothing is written to disk. A crash, hang or OOM cannot
// take the worker down. The worker container is the security boundary.
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

	// python has no flag like node's --max-old-space-size, so the harness sets RLIMIT_AS
	// itself from the env var pythonMemLimitEnv passes. It is higher than the node cap,
	// because RLIMIT_AS counts all address space, not only the heap.
	defaultPythonMemoryLimitMB = 512

	// killGracePeriod limits Wait after the process group is killed, so a child that ignores
	// SIGKILL (unlikely) cannot hang the node
	killGracePeriod = 2 * time.Second

	// stderrTailLimit: how much stderr we keep for errors. Only the tail matters, the
	// traceback ends there.
	stderrTailLimit = 64 * 1024
)

// tailBuffer keeps only the last limit bytes, so a subprocess that spams stderr cannot
// grow memory during its whole timeout
type tailBuffer struct {
	limit int
	buf   []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if n >= t.limit {
		t.buf = append(t.buf[:0], p[n-t.limit:]...)
		return n, nil
	}
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.limit; over > 0 {
		t.buf = t.buf[over:]
	}
	return n, nil
}

func (t *tailBuffer) String() string { return string(t.buf) }

// runSubprocess runs an interpreter in its own process group with a clean environment and
// the payload on stdin. On timeout the whole group gets SIGKILL, so grandchildren die too.
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

	var stdout bytes.Buffer
	stderr := &tailBuffer{limit: stderrTailLimit}
	cmd.Stdout = &stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		if runCtx.Err() != nil {
			return nil, fmt.Errorf("code execution timed out after %s", timeout)
		}
		return nil, fmt.Errorf("%s exited: %v\n%s", name, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// scrubbedEnv is the small environment that user code can see: PATH and the extras.
// DB and MQ credentials of the worker must never leak in.
func scrubbedEnv(extra ...string) []string {
	env := []string{"PATH=" + os.Getenv("PATH")}
	return append(env, extra...)
}

// pythonMemLimitEnv tells the python harnesses which RLIMIT_AS to set before the user code
// runs (see defaultPythonMemoryLimitMB).
func pythonMemLimitEnv(limitMB int) []string {
	return []string{fmt.Sprintf("YTSOAR_MEM_LIMIT_MB=%d", limitMB)}
}

// nodePathEnv passes NODE_PATH (the harness node_modules with nunjucks and the npm
// allowlist) when the worker has it.
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
