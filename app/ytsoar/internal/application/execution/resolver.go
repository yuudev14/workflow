package execution

import (
	"fmt"

	"github.com/yuudev14/ytsoar/internal/domain"
)

// StaticResolver routes tasks by connector id; connectors without an explicit
// mapping fall through to the default runtime.
type StaticResolver struct {
	defaultRuntime NodeRuntime
	byConnector    map[string]NodeRuntime
}

func NewStaticResolver(defaultRuntime NodeRuntime, byConnector map[string]NodeRuntime) *StaticResolver {
	if byConnector == nil {
		byConnector = map[string]NodeRuntime{}
	}
	return &StaticResolver{
		defaultRuntime: defaultRuntime,
		byConnector:    byConnector,
	}
}

func (r *StaticResolver) Resolve(task domain.Tasks) (NodeRuntime, error) {
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
