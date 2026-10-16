package execution

import (
	"fmt"

	"github.com/yuudev14/ytsoar/internal/domain"
)

// DynamicResolver chooses a runtime by calling your function each time a task arrives.
// It runs in the SANDBOX, behind the gRPC server. The worker's SandboxRuntime calls the
// sandbox over gRPC, and then this resolver picks the runtime that runs the task.
// The sandbox passes a function that reads info.json from the disk on every call, so a
// connector that you upload works with no restart.
// Compare StaticResolver, which uses a fixed list and runs in the worker.
type DynamicResolver func(task domain.Tasks) (TaskRuntime, error)

func (f DynamicResolver) Resolve(task domain.Tasks) (TaskRuntime, error) {
	return f(task)
}

// StaticResolver chooses a runtime from a fixed list that is made at start-up.
// It runs in the WORKER. The list holds the built-in connectors (http_request, condition),
// which run inside the worker (BuiltinRuntime). Every other connector goes to the default
// runtime, SandboxRuntime, which sends the task to the sandbox over gRPC.
// Compare DynamicResolver, which runs your code each time and runs in the sandbox.
type StaticResolver struct {
	defaultRuntime TaskRuntime
	byConnector    map[string]TaskRuntime
}

func NewStaticResolver(defaultRuntime TaskRuntime, byConnector map[string]TaskRuntime) *StaticResolver {
	if byConnector == nil {
		byConnector = map[string]TaskRuntime{}
	}
	return &StaticResolver{
		defaultRuntime: defaultRuntime,
		byConnector:    byConnector,
	}
}

func (r *StaticResolver) Resolve(task domain.Tasks) (TaskRuntime, error) {
	if task.ConnectorID != nil {
		if runtime, ok := r.byConnector[*task.ConnectorID]; ok {
			return runtime, nil
		}
	}
	if r.defaultRuntime == nil {
		return nil, fmt.Errorf("no runtime registered for connector %v", task.ConnectorID)
	}
	return r.defaultRuntime, nil
}
