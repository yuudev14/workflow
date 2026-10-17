package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/application/playbooks"
	"github.com/yuudev14/ytsoar/internal/application/tasks"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
	"github.com/yuudev14/ytsoar/internal/types"
)

func status(s string) types.Nullable[string] { return types.Nullable[string]{Value: &s, Set: true} }

// The guard is in SQL, so this test needs a migrated Postgres:
// TEST_DATABASE_URL=postgres://... go test ./internal/adapters/repository
func TestStatusTransitionsAreGuardedInSQL(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	require.NoError(t, err)
	defer pool.Close()
	queries := db.New(pool)
	playbookRepo := NewPlaybookRepository(logger.NewNop(), queries, pool)
	taskRepo := NewTaskRepositoryImpl(logger.NewNop(), queries, pool)

	var playbookID, taskID, historyID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO playbooks (name) VALUES ('transitions') RETURNING id`).Scan(&playbookID))
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM playbooks WHERE id = $1`, playbookID) })
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO tasks (playbook_id, name, description, parameters, operation) VALUES ($1, 'a', '', '{}', 'op') RETURNING id`, playbookID).Scan(&taskID))
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO playbook_history (playbook_id, triggered_at, edges, status) VALUES ($1, NOW(), '[]', 'queued') RETURNING id`, playbookID).Scan(&historyID))
	_, err = pool.Exec(ctx,
		`INSERT INTO task_history (playbook_history_id, task_id, triggered_at, name, description, parameters, operation) VALUES ($1, $2, NOW(), 'a', '', '{}', 'op')`, historyID, taskID)
	require.NoError(t, err)

	run := func(to string) error {
		_, err := playbookRepo.UpdatePlaybookHistory(ctx, historyID.String(), playbooks.UpdatePlaybookHistoryData{Status: status(to)})
		return err
	}
	task := func(to string) error {
		_, err := taskRepo.UpdateTaskHistory(ctx, historyID.String(), taskID.String(), tasks.UpdateTaskHistoryData{Name: "a", Operation: "op", Status: status(to)})
		return err
	}

	// A run starts one time. The second worker that claims it gets a refusal.
	assert.NoError(t, run(domain.StatusInProgress))
	assert.ErrorIs(t, run(domain.StatusInProgress), domain.ErrInvalidTransition)
	assert.NoError(t, run(domain.StatusSuccess))
	// A finished run does not change again.
	assert.ErrorIs(t, run(domain.StatusFailed), domain.ErrInvalidTransition)
	_, err = playbookRepo.UpdatePlaybookHistoryStatus(ctx, historyID.String(), domain.StatusInProgress)
	assert.ErrorIs(t, err, domain.ErrInvalidTransition)

	assert.NoError(t, task(domain.StatusInProgress))
	assert.NoError(t, task(domain.StatusSuccess))
	assert.ErrorIs(t, task(domain.StatusFailed), domain.ErrInvalidTransition)

	// Crash recovery fails the tasks that did not finish. It keeps the tasks that did finish.
	_, err = pool.Exec(ctx, `INSERT INTO task_history (playbook_history_id, task_id, triggered_at, name, description, parameters, operation, status)
		SELECT $1, $2, NOW(), 'b', '', '{}', 'op', 'in_progress'`, historyID, taskID)
	require.NoError(t, err)
	failed, err := taskRepo.FailUnfinishedTaskHistory(ctx, historyID.String(), "worker stopped")
	require.NoError(t, err)
	assert.Len(t, failed, 1, "it returns the tasks that it changed, so the caller can send the events")
	var statuses []string
	rows, err := pool.Query(ctx, `SELECT status::text FROM task_history WHERE playbook_history_id = $1 ORDER BY name`, historyID)
	require.NoError(t, err)
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		statuses = append(statuses, s)
	}
	assert.Equal(t, []string{"success", "failed"}, statuses)
}

// The worker builds the message of a run from this row. It needs the trigger input.
func TestGetPlaybookHistoryByIdReturnsTheTriggerInput(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	require.NoError(t, err)
	defer pool.Close()
	playbookRepo := NewPlaybookRepository(logger.NewNop(), db.New(pool), pool)

	var playbookID, historyID uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO playbooks (name) VALUES ('input') RETURNING id`).Scan(&playbookID))
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM playbooks WHERE id = $1`, playbookID) })
	require.NoError(t, pool.QueryRow(ctx,
		`INSERT INTO playbook_history (playbook_id, triggered_at, edges, status, trigger_type, input)
		 VALUES ($1, NOW(), '[]', 'queued', 'manual', '{"module_type":"alert","records":[{"id":"r1"}]}') RETURNING id`, playbookID).Scan(&historyID))

	history, err := playbookRepo.GetPlaybookHistoryById(ctx, historyID)

	require.NoError(t, err)
	assert.JSONEq(t, `{"module_type":"alert","records":[{"id":"r1"}]}`, string(history.Input))
	if assert.NotNil(t, history.TriggerType) {
		assert.Equal(t, "manual", *history.TriggerType)
	}
}
