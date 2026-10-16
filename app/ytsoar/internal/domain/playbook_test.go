package domain

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Finished statuses must never be a source: nothing moves a finished row.
func TestTransitionsAreForwardOnly(t *testing.T) {
	finished := []string{StatusSuccess, StatusFailed, StatusSkipped}
	for name, table := range map[string]Transitions{"run": RunTransitions, "task": TaskTransitions} {
		for to, froms := range table {
			assert.NotContains(t, froms, to, "%s: %s -> itself", name, to)
			for _, from := range froms {
				assert.False(t, slices.Contains(finished, from), "%s: %s is finished but moves to %s", name, from, to)
			}
		}
	}
}

func TestTransitionsFrom(t *testing.T) {
	assert.Equal(t, []string{StatusQueued}, RunTransitions.From(StatusInProgress))
	assert.Empty(t, RunTransitions.From(StatusQueued), "nothing moves back to queued")
	assert.Empty(t, RunTransitions.From("unknown"))
}
