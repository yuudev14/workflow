package repository_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/yuudev14-workflow/workflow-service/dto"
	"github.com/yuudev14-workflow/workflow-service/internal/repository"
	"github.com/yuudev14-workflow/workflow-service/internal/utils"
	"github.com/yuudev14-workflow/workflow-service/models"
)

func TestGetWorkflows(t *testing.T) {
	_, sqlxDB, mock := utils.SetupMockEnvironment(t)
	repo := repository.NewWorkflowRepository(sqlxDB)
	assert.Equal(t, 1, 1)

	// Test data
	workflows := []models.Workflows{
		{Name: "Workflow A", UpdatedAt: time.Now()},
		{Name: "Workflow B", UpdatedAt: time.Now().Add(-time.Hour)},
	}

	rows := sqlmock.NewRows([]string{
		"name", "updated_at",
	})

	for _, wf := range workflows {
		rows.AddRow(
			wf.Name,
			wf.UpdatedAt,
		)
	}

	// Expected query patterns (regex matches sq-generated SQL)
	expectedQueryNoFilter := `SELECT \* FROM workflows.*ORDER BY updated_at DESC.*LIMIT \d+ OFFSET \d+`
	// expectedQueryWithFilter := `SELECT \* FROM workflows WHERE name ILIKE '\%Workflow A\%' ORDER BY updated_at DESC.*LIMIT \d+ OFFSET \d+`
	expectedQueryWithFilter := `SELECT \* FROM workflows WHERE name ILIKE \? ORDER BY updated_at DESC.*LIMIT \d+ OFFSET \d+`

	tests := []struct {
		name      string
		offset    int
		limit     int
		filter    dto.WorkflowFilter
		mockQuery string
		wantLen   int
		wantErr   bool
	}{
		{
			name:      "no filter, offset 0, limit 2",
			offset:    0,
			limit:     2,
			filter:    dto.WorkflowFilter{},
			mockQuery: expectedQueryNoFilter,
		},
		{
			name:      "with name filter",
			offset:    0,
			limit:     10,
			filter:    dto.WorkflowFilter{Name: utils.StrPtr("Workflow A")},
			mockQuery: expectedQueryWithFilter,
		},
		{
			name:      "offset 2, limit 1",
			offset:    2,
			limit:     1,
			filter:    dto.WorkflowFilter{},
			mockQuery: expectedQueryNoFilter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery(tt.mockQuery).
				WillReturnRows(rows)

			_, err := repo.GetWorkflows(tt.offset, tt.limit, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}

}
