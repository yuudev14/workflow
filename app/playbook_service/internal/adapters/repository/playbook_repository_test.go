package repository

// import (
// 	"testing"
// 	"time"

// 	"github.com/DATA-DOG/go-sqlmock"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/yuudev14/ytsoar/internal/utils"
// 	"github.com/yuudev14/ytsoar/internal/application/playbooks"
// )

// func TestGetPlaybooks(t *testing.T) {
// 	_, sqlxDB, mock := utils.SetupMockEnvironment(t)
// 	repo := workflows.NewPlaybookRepository(sqlxDB)
// 	assert.Equal(t, 1, 1)

// 	// Test data
// 	workflowDatas := []domain.Playbooks{
// 		{Name: "Playbook A", UpdatedAt: time.Now()},
// 		{Name: "Playbook B", UpdatedAt: time.Now().Add(-time.Hour)},
// 	}

// 	rows := sqlmock.NewRows([]string{
// 		"name", "updated_at",
// 	})

// 	for _, wf := range workflowDatas {
// 		rows.AddRow(
// 			wf.Name,
// 			wf.UpdatedAt,
// 		)
// 	}

// 	// Expected query patterns (regex matches sq-generated SQL)
// 	expectedQueryNoFilter := `SELECT \* FROM workflows.*ORDER BY updated_at DESC.*LIMIT \d+ OFFSET \d+`
// 	// expectedQueryWithFilter := `SELECT \* FROM workflows WHERE name ILIKE '\%Playbook A\%' ORDER BY updated_at DESC.*LIMIT \d+ OFFSET \d+`
// 	expectedQueryWithFilter := `SELECT \* FROM workflows WHERE name ILIKE \? ORDER BY updated_at DESC.*LIMIT \d+ OFFSET \d+`

// 	tests := []struct {
// 		name      string
// 		offset    int
// 		limit     int
// 		filter    workflows.PlaybookFilter
// 		mockQuery string
// 		wantLen   int
// 		wantErr   bool
// 	}{
// 		{
// 			name:      "no filter, offset 0, limit 2",
// 			offset:    0,
// 			limit:     2,
// 			filter:    workflows.PlaybookFilter{},
// 			mockQuery: expectedQueryNoFilter,
// 		},
// 		{
// 			name:      "with name filter",
// 			offset:    0,
// 			limit:     10,
// 			filter:    workflows.PlaybookFilter{Name: utils.StrPtr("Playbook A")},
// 			mockQuery: expectedQueryWithFilter,
// 		},
// 		{
// 			name:      "offset 2, limit 1",
// 			offset:    2,
// 			limit:     1,
// 			filter:    workflows.PlaybookFilter{},
// 			mockQuery: expectedQueryNoFilter,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mock.ExpectQuery(tt.mockQuery).
// 				WillReturnRows(rows)

// 			_, err := repo.GetPlaybooks(tt.offset, tt.limit, tt.filter)

// 			if tt.wantErr {
// 				assert.Error(t, err)
// 				return
// 			}
// 			assert.NoError(t, err)
// 		})
// 	}

// }
