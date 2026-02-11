package workflows_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/internal/workflows"
)

func TestAcyclicalGraphs(t *testing.T) {

	tests := []struct {
		graph    map[string][]string
		expected bool
	}{
		{
			graph: map[string][]string{
				"A": {"B", "C"},
				"B": {"D"},
				"C": {"E"},
				"D": {},
				"E": {"F"},
				"F": {},
			},
			expected: false,
		},
		{
			graph: map[string][]string{
				"A": {"B", "C"},
				"B": {"D", "A"},
				"C": {"E", "D"},
				"D": {},
				"E": {"F"},
				"F": {},
			},
			expected: true,
		},
		{
			graph: map[string][]string{
				"A": {"B", "C"},
				"B": {"D"},
				"C": {"E", "D"},
				"D": {},
				"E": {"F"},
				"F": {},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		assert.Equal(t, workflows.IsAcyclicGraph(test.graph), test.expected)
	}
}
