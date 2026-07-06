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

// StatusPublisher emits playbook/task status events from the worker to the
// API process (fanout exchange -> WS hub). data is the updated history row,
// the same struct the hub broadcasts today.
type StatusPublisher interface {
	Publish(event string, data any) error
}
