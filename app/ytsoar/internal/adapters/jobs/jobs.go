// Package jobs runs playbook runs on River. River is a job queue in Postgres.
// The API queues a run in the same transaction that creates its history rows.
// So the system cannot lose the run, and it cannot queue the run without its history.
package jobs

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

const (
	queueName = "playbook"
	// maxAttempts matters in two cases. In the first case, the system cannot build the message,
	// and a retry can help. In the second case, a worker stopped in the middle of a run.
	// Then the second attempt marks the run failed. It does not start the run again.
	// The system never retries a run that failed.
	maxAttempts = 5
	// rescueGrace is the time after the run timeout. After this time, River decides that the
	// worker of a running job stopped. It must be longer than the run timeout.
	// If it is shorter, River takes a live run from its worker.
	rescueGrace = 5 * time.Minute
)

// RunArgs is the job payload. It holds only the run ID. The worker builds the rest
// from the history rows. So river_job stays small at high volume.
type RunArgs struct {
	PlaybookHistoryID uuid.UUID `json:"playbook_history_id"`
}

func (RunArgs) Kind() string { return "playbook_run" }

func (RunArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: queueName, MaxAttempts: maxAttempts}
}

// MessageLoader builds the message of a queued run. playbooks.RunLoader implements it.
type MessageLoader interface {
	LoadRunMessage(ctx context.Context, playbookHistoryID uuid.UUID) (domain.TaskMessage, error)
}

// RunExecutor runs a playbook. execution.Executor implements it.
type RunExecutor interface {
	Run(ctx context.Context, msg domain.TaskMessage) error
	Abort(ctx context.Context, playbookHistoryID uuid.UUID, cause error) error
}

type RunWorker struct {
	river.WorkerDefaults[RunArgs]
	logger   logger.Logger
	loader   MessageLoader
	executor RunExecutor
}

func NewRunWorker(log logger.Logger, loader MessageLoader, executor RunExecutor) *RunWorker {
	return &RunWorker{logger: log, loader: loader, executor: executor}
}

func (w *RunWorker) Work(ctx context.Context, job *river.Job[RunArgs]) error {
	id := job.Args.PlaybookHistoryID
	msg, err := w.loader.LoadRunMessage(ctx, id)
	if err == nil {
		err = w.executor.Run(ctx, msg)
		if !errors.Is(err, execution.ErrNotStarted) {
			// The executor already saved a failed run, so a failed run is not a job error.
			// If the worker returned the error, River would retry the job and run the connectors again.
			if err != nil {
				w.logger.Errorw("playbook run failed", "playbook_history_id", id, "error", err)
			}
			return nil
		}
	}

	// The system could not build or start the run, and nothing ran.
	// River retries the job after a delay. On the last attempt, the worker marks the run failed.
	// Without this, the run stays queued forever.
	if job.Attempt < job.MaxAttempts {
		return err
	}
	w.logger.Errorw("could not start the run, failing it", "playbook_history_id", id, "error", err)
	return w.executor.Abort(ctx, id, err)
}

// Enqueuer queues runs. It implements contracts.RunEnqueuer.
type Enqueuer struct {
	client *river.Client[pgx.Tx]
	// txFrom finds the transaction that a TxManager put in ctx. main.go injects it,
	// so this adapter does not import the repository adapter.
	txFrom func(context.Context) (pgx.Tx, bool)
}

// NewEnqueuer is for the API. It makes an insert-only River client.
// The client queues runs and does not work them. The worker process works them.
func NewEnqueuer(pool *pgxpool.Pool, txFrom func(context.Context) (pgx.Tx, bool)) (*Enqueuer, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return nil, err
	}
	return &Enqueuer{client: client, txFrom: txFrom}, nil
}

// EnqueueRun joins the transaction in ctx, if ctx has one.
func (e *Enqueuer) EnqueueRun(ctx context.Context, playbookHistoryID uuid.UUID) error {
	args := RunArgs{PlaybookHistoryID: playbookHistoryID}
	var err error
	if tx, ok := e.txFrom(ctx); ok {
		_, err = e.client.InsertTx(ctx, tx, args, nil)
	} else {
		_, err = e.client.Insert(ctx, args, nil)
	}
	return err
}

// NewWorkerClient works playbook runs. It works `concurrency` runs at the same time.
// runTimeout is the time limit for one run.
func NewWorkerClient(pool *pgxpool.Pool, worker *RunWorker, concurrency int, runTimeout time.Duration) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:               map[string]river.QueueConfig{queueName: {MaxWorkers: max(concurrency, 1)}},
		Workers:              workers,
		JobTimeout:           runTimeout,
		RescueStuckJobsAfter: runTimeout + rescueGrace,
	})
}

// ModuleEventLogger implements contracts.ModuleEventPublisher. Nothing uses alert and
// incident events yet, so it only writes them to the log. Event-triggered playbooks
// (on_create, on_update) will queue their runs from here.
type ModuleEventLogger struct{ logger logger.Logger }

func NewModuleEventLogger(log logger.Logger) *ModuleEventLogger {
	return &ModuleEventLogger{logger: log}
}

func (l *ModuleEventLogger) Publish(module, event string, _ any) error {
	l.logger.Debugw("module event", "module", module, "event", event)
	return nil
}
