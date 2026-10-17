package execution

import (
	"fmt"

	"github.com/yuudev14/ytsoar/internal/domain"
)

// DynamicResolver picks a runtime by calling your function each time a task arrives. It
// runs in the SANDBOX, behind the gRPC server. SandboxRuntime in the worker calls the
// sandbox over gRPC, and this resolver picks the runtime for the task. The function reads
// info.json from disk on every call, so an uploaded connector works without a restart.
// Compare StaticResolver (fixed list, runs in the worker).
type DynamicResolver func(task domain.Tasks) (TaskRuntime, error)

func (f DynamicResolver) Resolve(task domain.Tasks) (TaskRuntime, error) {
	return f(task)
}

// StaticResolver picks a runtime from a fixed list made at start-up. It runs in the
// WORKER. The list has the built-in connectors (http_request, condition), which run in the
// worker (BuiltinRuntime). All other connectors go to the default runtime, SandboxRuntime,
// which sends the task to the sandbox over gRPC. Compare DynamicResolver (runs in the
// sandbox).
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
