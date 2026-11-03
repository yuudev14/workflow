package execution

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/execution_mock.go -package=mocks . TaskRuntime,RuntimeResolver,StatusPublisher,ConfigSource

// ExecutionRequest has everything a runtime needs to run one playbook node. Steps is a
// snapshot of the outputs of earlier nodes by task name. The runtime does the templating.
type ExecutionRequest struct {
	Task              domain.Tasks
	Steps             map[string]any
	PlaybookHistoryID uuid.UUID
	Timeout           time.Duration
	// Input is the trigger data of the run, available to templates as `var.input`. It is
	// never nil, so a template that uses it does not raise on a run without trigger data.
	Input *domain.RunInput
	// Config is the plain connector config from the database.
	Config map[string]any
}

// TaskRuntime runs one task (one playbook node) and returns its JSON result. Every type
// that ends with "Runtime" fits it:
//   - BuiltinRuntime: the built-in connectors. Runs in the WORKER.
//   - SandboxRuntime: sends the task to the sandbox over gRPC. Runs in the WORKER.
//   - PythonSnippetRuntime, NodeJSSnippetRuntime, PythonConnectorRuntime,
//     NodeJSConnectorRuntime: run user code in a new process. Run in the SANDBOX.
type TaskRuntime interface {
	Execute(ctx context.Context, req ExecutionRequest) (json.RawMessage, error)
}

// RuntimeResolver answers one question: which runtime runs this task? Two types fit it:
//   - StaticResolver: a fixed list. Runs in the WORKER.
//   - DynamicResolver: your function, called each time. Runs in the SANDBOX behind the
//     gRPC server.
type RuntimeResolver interface {
	Resolve(task domain.Tasks) (TaskRuntime, error)
}

// StatusPublisher tells every API replica that a run row or a task row changed. The event
// is task_status or playbook_status, and id is the history row. The API loads the row and
// sends it to its WebSocket clients, so the payload stays small.
type StatusPublisher interface {
	Publish(ctx context.Context, event string, id uuid.UUID) error
}

// ConfigSource gives the plain values of a connector config for a run.
type ConfigSource interface {
	RunConfig(ctx context.Context, connectorID string, name *string) (map[string]any, error)
}
