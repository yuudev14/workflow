package execution

import "sync"

// StepStore collects the node outputs during a run. A snapshot goes with every runtime
// call, so templates can use earlier steps.
//
// It is copy-on-write: Set clones the map, Snapshot returns the current immutable one.
// Snapshot runs once per node (also for nodes that never use steps), Set only for nodes
// with output. So the copy cost is on the rarer path and Snapshot stays O(1). The returned
// map is shared, callers must treat it as read-only.
type StepStore struct {
	mu    sync.Mutex
	steps map[string]any
}

func NewStepStore() *StepStore {
	return &StepStore{steps: map[string]any{}}
}

func (s *StepStore) Set(node string, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make(map[string]any, len(s.steps)+1)
	for k, v := range s.steps {
		next[k] = v
	}
	next[node] = result
	s.steps = next
}

// Snapshot returns the current step map. It is shared and must not be mutated.
func (s *StepStore) Snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.steps
}
