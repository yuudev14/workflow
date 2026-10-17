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
	runErr      error
	abortErr    error
	ran         chan uuid.UUID
	aborted     []uuid.UUID
	abortCauses []error
}

func (f *fakeExecutor) Run(_ context.Context, msg domain.TaskMessage) error {
	if f.ran != nil {
		f.ran <- msg.PlaybookHistoryId
	}
	return f.runErr
}

func (f *fakeExecutor) Abort(_ context.Context, id uuid.UUID, cause error) error {
	f.aborted = append(f.aborted, id)
	f.abortCauses = append(f.abortCauses, cause)
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

// The executor already saved a failed run. If the worker returned the error, River would
// retry the job and run the connectors again.
func TestWorkDoesNotRetryAFailedRun(t *testing.T) {
	exec := &fakeExecutor{runErr: errors.New("connector failed"), ran: make(chan uuid.UUID, 1)}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(uuid.New(), 1)))
}

// Nothing ran, because the run could not start. The worker marks the run failed at once
// and returns no error, so River does not retry.
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

// On attempt 2, the first attempt may have run the playbook already. The error text of the
// run must not say that the run could not start.
func TestWorkSaysAnEarlierAttemptDidNotFinishOnAttempt2(t *testing.T) {
	exec := &fakeExecutor{}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{err: errors.New("connection refused")}, exec)

	assert.NoError(t, worker.Work(context.Background(), job(uuid.New(), 2)))
	if assert.Len(t, exec.abortCauses, 1) {
		assert.ErrorContains(t, exec.abortCauses[0], "an earlier attempt did not finish this run")
		assert.ErrorContains(t, exec.abortCauses[0], "connection refused")
	}
}

// The worker is stopping. The run must not be failed: River puts the job back, and a new
// worker starts the run.
func TestWorkDoesNotFailTheRunWhenTheContextIsCancelled(t *testing.T) {
	exec := &fakeExecutor{}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{err: context.Canceled}, exec)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.Error(t, worker.Work(ctx, job(uuid.New(), 1)))
	assert.Empty(t, exec.aborted)
}

// If Abort fails too (database down), the error goes to River and attempt 2 tries again.
func TestWorkReturnsTheErrorWhenAbortFails(t *testing.T) {
	exec := &fakeExecutor{runErr: execution.ErrNotStarted, abortErr: errors.New("db down")}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.Error(t, worker.Work(context.Background(), job(uuid.New(), 1)))
}

// The run ran, but its final status was not saved. The worker returns the error, so River
// gives the job to a new attempt, which marks the run failed. The worker does not abort.
func TestWorkRetriesWhenTheFinalStatusWasNotSaved(t *testing.T) {
	exec := &fakeExecutor{runErr: execution.ErrNotFinished}
	worker := NewRunWorker(logger.NewNop(), fakeLoader{}, exec)

	assert.ErrorIs(t, worker.Work(context.Background(), job(uuid.New(), 1)), execution.ErrNotFinished)
	assert.Empty(t, exec.aborted)
}

// This test needs a migrated Postgres: TEST_DATABASE_URL=postgres://... go test
// ./internal/adapters/jobs. Use a throwaway database: the test starts a worker on the
// "playbook" queue, and it would also take real jobs.
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
	// the test removes only the jobs that it made
	rolledBackID, id := uuid.New(), uuid.New()
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM river_job WHERE args->>'playbook_history_id' = ANY($1)", []string{rolledBackID.String(), id.String()})
	})

	// a rolled back transaction removes its job
	rollback := errors.New("rollback")
	err = txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		require.NoError(t, enqueuer.EnqueueRun(ctx, rolledBackID))
		return rollback
	})
	require.ErrorIs(t, err, rollback)
	assert.Equal(t, 0, count(rolledBackID))

	// a committed transaction keeps its job, and the worker works it
	exec := &fakeExecutor{ran: make(chan uuid.UUID, 1)}
	require.NoError(t, txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		return enqueuer.EnqueueRun(ctx, id)
	}))
	assert.Equal(t, 1, count(id))

	workerClient, err := NewWorkerClient(pool, NewRunWorker(logger.NewNop(), fakeLoader{}, exec), NewRunForEventWorker(logger.NewNop(), nil), 1, time.Minute, time.Second)
	require.NoError(t, err)
	require.NoError(t, workerClient.Start(ctx))
	defer func() { _ = workerClient.Stop(ctx) }()
	// other tests can leave queued runs in this database, and the worker works them too
	timeout := time.After(10 * time.Second)
	for {
		select {
		case got := <-exec.ran:
			if got == id {
				return
			}
		case <-timeout:
			t.Fatal("the committed run was never worked")
		}
	}
}
