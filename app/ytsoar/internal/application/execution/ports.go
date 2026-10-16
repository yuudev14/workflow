package execution

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/execution_mock.go -package=mocks . NodeRuntime,RuntimeResolver,StatusPublisher

// ExecutionRequest carries everything a runtime needs to execute one playbook
// node. Steps is a snapshot of prior node outputs keyed by task name;
// templating against it happens inside the runtime, not here.
type ExecutionRequest struct {
	Task              domain.Tasks
	Steps             map[string]any
	PlaybookHistoryID uuid.UUID
	Timeout           time.Duration
	// Input is the run's trigger data, exposed to templates as `var.input`. It
	// is always non-nil so a template referencing it never raises on a run that
	// had no trigger data.
	Input *domain.RunInput
}

// NodeRuntime executes one playbook node and returns its JSON-encoded result.
// Implementations: grpcruntime (worker side, dials the sandbox), localexec
// subprocess runners (sandbox side), goconnectors registry (final phase).
type NodeRuntime interface {
	Execute(ctx context.Context, req ExecutionRequest) (json.RawMessage, error)
}

// RuntimeResolver picks the runtime responsible for a task.
type RuntimeResolver interface {
	Resolve(task domain.Tasks) (NodeRuntime, error)
}

// StatusPublisher tells every API replica that a run row or a task row changed.
// The event is task_status or playbook_status, and id is the history row. The API loads
// the row and sends it to its WebSocket clients. So the payload stays small, also for
// a big task result.
type StatusPublisher interface {
	Publish(ctx context.Context, event string, id uuid.UUID) error
}
