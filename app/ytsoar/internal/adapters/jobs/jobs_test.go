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
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

type fakeLoader struct{ err error }

func (f fakeLoader) LoadRunMessage(_ context.Context, id uuid.UUID) (domain.TaskMessage, error) {
	return domain.TaskMessage{PlaybookHistoryId: id}, f.err
}

type fakeExecutor struct {
	runErr  error
	ran     chan uuid.UUID
	aborted []uuid.UUID
}

func (f *fakeExecutor) Run(_ context.Context, msg domain.TaskMessage) error {
	if f.ran != nil {
		f.ran <- msg.PlaybookHistoryId
	}
	return f.runErr
}

func (f *fakeExecutor) Abort(_ context.Context, id uuid.UUID, _ error) error {
	f.aborted = append(f.aborted, id)
	return nil
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

// A failed run is already persisted by the executor. Returning it would make
// River retry the job and run the connectors a second time.
func TestWorkDoesNotRetryAFailedRun(t *testing.T) {
	exec := &fakeExecutor{runErr: errors.New("connector failed"), ran: make(chan uuid.UUID, 1)}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(uuid.New(), 1)))
}

func TestWorkRetriesWhenTheMessageCannotBeLoaded(t *testing.T) {
	exec := &fakeExecutor{}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{err: errors.New("db down")}, exec)

	assert.Error(t, worker.Work(context.Background(), job(uuid.New(), 1)))
	assert.Empty(t, exec.aborted)
}

// On the last attempt the run is failed, or it would stay queued forever.
func TestWorkFailsTheRunOnTheLastAttempt(t *testing.T) {
	id, exec := uuid.New(), &fakeExecutor{}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{err: errors.New("db down")}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(id, maxAttempts)))
	assert.Equal(t, []uuid.UUID{id}, exec.aborted)
}

// Needs a migrated Postgres: TEST_DATABASE_URL=postgres://... go test ./internal/adapters/jobs
func TestEnqueueRunIsTransactional(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx, url)
	require.NoError(t, err)
	defer pool.Close()
	_, err = pool.Exec(ctx, "TRUNCATE river_job")
	require.NoError(t, err)

	insertClient, err := NewInsertClient(pool)
	require.NoError(t, err)
	enqueuer, txManager := NewEnqueuer(insertClient), repository.NewPgxTxManager(pool)
	count := func() (n int) {
		require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&n))
		return n
	}

	// a rolled back transaction takes its job with it
	rollback := errors.New("rollback")
	err = txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		require.NoError(t, enqueuer.EnqueueRun(ctx, uuid.New()))
		return rollback
	})
	require.ErrorIs(t, err, rollback)
	assert.Equal(t, 0, count())

	// a committed one is worked
	id, exec := uuid.New(), &fakeExecutor{ran: make(chan uuid.UUID, 1)}
	require.NoError(t, txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		return enqueuer.EnqueueRun(ctx, id)
	}))
	assert.Equal(t, 1, count())

	workerClient, err := NewWorkerClient(pool, NewRunWorker(logger.NewNop(), fakeLoader{}, exec), 1, time.Minute)
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
