package playbooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
)

func TestValidatePlaybookTaskPayload(t *testing.T) {
	tests := []struct {
		name      string
		edges     map[string][]string
		nodes     []tasks.TaskPayload
		withError bool
	}{
		{
			name: "with no error",
			edges: map[string][]string{
				"start": {"task1"},
			},
			nodes: []tasks.TaskPayload{
				{Name: "start"},
				{Name: "task1"},
			},
			withError: false,
		},

		{
			name:  "no start in edges",
			edges: map[string][]string{},
			nodes: []tasks.TaskPayload{
				{Name: "start"},
				{Name: "task1"},
			},
			withError: true,
		},
		{
			name: "no start in edges",
			edges: map[string][]string{
				"start": {"task1"},
			},
			nodes:     []tasks.TaskPayload{},
			withError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := UpdatePlaybookTasksPayload{
				Edges: tt.edges,
				Nodes: tt.nodes,
			}

			err := validatePlaybookTaskPayload(body)

			if tt.withError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}
