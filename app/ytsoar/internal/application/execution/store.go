package execution

import "sync"

// StepStore accumulates node outputs during a run. A snapshot rides along
// with every runtime call so templates can reference prior steps.
//
// It is copy-on-write: Set clones the map, Snapshot just returns the current
// immutable one. Snapshot is taken once per node (including nodes that never
// reference steps), Set only for nodes that produce output, so the copy cost
// lives on the rarer path and Snapshot stays O(1). The returned map is shared
// and must be treated as read-only by callers.
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
