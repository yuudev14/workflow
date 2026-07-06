package execution

import "sync"

// StepStore accumulates node outputs during a run. A snapshot rides along
// with every runtime call so templates can reference prior steps.
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
	s.steps[node] = result
}

func (s *StepStore) Snapshot() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]any, len(s.steps))
	for k, v := range s.steps {
		out[k] = v
	}
	return out
}
