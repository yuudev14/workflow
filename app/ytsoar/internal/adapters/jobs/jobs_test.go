package jobs

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuudev14/ytsoar/db"
	"github.com/yuudev14/ytsoar/internal/adapters/repository"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type fakeLoader struct{ err error }

func (f fakeLoader) LoadRunMessage(_ context.Context, id uuid.UUID) (domain.TaskMessage, error) {
	return domain.TaskMessage{PlaybookHistoryId: id}, f.err
}

type fakeExecutor struct {
	runErr   error
	abortErr error
	ran      chan uuid.UUID
	aborted  []uuid.UUID
}

func (f *fakeExecutor) Run(_ context.Context, msg domain.TaskMessage) error {
	if f.ran != nil {
		f.ran <- msg.PlaybookHistoryId
	}
	return f.runErr
}

func (f *fakeExecutor) Abort(_ context.Context, id uuid.UUID, _ error) error {
	f.aborted = append(f.aborted, id)
	return f.abortErr
}

func job(id uuid.UUID, attempt int) *river.Job[RunArgs] {
	return &river.Job[RunArgs]{JobRow: &rivertype.JobRow{Attempt: attempt, MaxAttempts: maxAttempts}, Args: RunArgs{id}}
}

func TestWorkRunsTheLoadedMessage(t *testing.T) {
	id, exec := uuid.New(), &fakeExecutor{ran: make(chan uuid.UUID, 1)}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(id, 1)))
	assert.Equal(t, id, <-exec.ran)
}

// The executor already saved a failed run. If the worker returned the error,
// River would retry the job and run the connectors a second time.
func TestWorkDoesNotRetryAFailedRun(t *testing.T) {
	exec := &fakeExecutor{runErr: errors.New("connector failed"), ran: make(chan uuid.UUID, 1)}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(uuid.New(), 1)))
}

// Nothing ran when the run could not start. The worker marks the run failed at once, and it
// does not return an error, so River does not retry.
func TestWorkFailsARunThatDidNotStartAtOnce(t *testing.T) {
	id, exec := uuid.New(), &fakeExecutor{runErr: execution.ErrNotStarted}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(id, 1)))
	assert.Equal(t, []uuid.UUID{id}, exec.aborted)
}

func TestWorkFailsTheRunWhenTheMessageCannotBeLoaded(t *testing.T) {
	id, exec := uuid.New(), &fakeExecutor{}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{err: errors.New("db down")}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(id, 1)))
	assert.Equal(t, []uuid.UUID{id}, exec.aborted)
}

// If Abort fails too (the database is down), the error goes to River, so the second attempt tries again.
func TestWorkReturnsTheErrorWhenAbortFails(t *testing.T) {
	exec := &fakeExecutor{runErr: execution.ErrNotStarted, abortErr: errors.New("db down")}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.Error(t, worker.Work(context.Background(), job(uuid.New(), 1)))
}

// The run ran, but its final status was not saved. The worker returns the error, so River gives
// the job to a new attempt. That attempt marks the run failed. The worker does not abort here.
func TestWorkRetriesWhenTheFinalStatusWasNotSaved(t *testing.T) {
	exec := &fakeExecutor{runErr: execution.ErrNotFinished}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.ErrorIs(t, worker.Work(context.Background(), job(uuid.New(), 1)), execution.ErrNotFinished)
	assert.Empty(t, exec.aborted)
}

// This test needs a migrated Postgres: TEST_DATABASE_URL=postgres://... go test ./internal/adapters/jobs
// Use a throwaway database. The test starts a worker on the "playbook" queue, and that worker
// would also take real jobs.
func TestEnqueueRunIsTransactional(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	require.NoError(t, err)
	defer pool.Close()

	enqueuer, err := NewEnqueuer(pool, repository.TxFromContext)
	require.NoError(t, err)
	txManager := repository.NewPgxTxManager(pool)
	count := func(id uuid.UUID) (n int) {
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM river_job WHERE args->>'playbook_history_id' = $1", id.String()).Scan(&n))
		return n
	}
	// The test removes only the jobs that it made.
	rolledBackID, id := uuid.New(), uuid.New()
	t.Cleanup(func() {
		pool.Exec(ctx, "DELETE FROM river_job WHERE args->>'playbook_history_id' = ANY($1)", []string{rolledBackID.String(), id.String()})
	})

	// A rolled back transaction removes its job.
	rollback := errors.New("rollback")
	err = txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		require.NoError(t, enqueuer.EnqueueRun(ctx, rolledBackID))
		return rollback
	})
	require.ErrorIs(t, err, rollback)
	assert.Equal(t, 0, count(rolledBackID))

	// A committed transaction keeps its job, and the worker works it.
	exec := &fakeExecutor{ran: make(chan uuid.UUID, 1)}
	require.NoError(t, txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		return enqueuer.EnqueueRun(ctx, id)
	}))
	assert.Equal(t, 1, count(id))

	workerClient, err := NewWorkerClient(pool, NewRunWorker(logger.NewNop(), fakeLoader{}, exec), 1, time.Minute, time.Second)
	require.NoError(t, err)
	require.NoError(t, workerClient.Start(ctx))
	defer workerClient.Stop(ctx)
	select {
	case got := <-exec.ran:
		assert.Equal(t, id, got)
	case <-time.After(10 * time.Second):
		t.Fatal("the committed run was never worked")
	}
}
