// Package jobs runs playbook runs on River, a job queue stored in Postgres. A
// run is queued in the same transaction that creates its history rows, so it
// can neither be lost nor queued without its history.
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
	"github.com/yuudev14/ytsoar/internal/adapters/repository"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

const (
	queueName = "playbook"
	// maxAttempts only matters when the message cannot be rebuilt (a retry may
	// help) or a worker died mid-run (the second attempt marks the run failed
	// instead of running it again). A run that failed is never retried.
	maxAttempts = 5
	// rescueGrace is how long past the run timeout a running job may sit before
	// River decides its worker died. It must exceed the run timeout, or a live
	// run would be taken from its worker.
	rescueGrace = 5 * time.Minute
)

// RunArgs is the job payload: only the run id. The worker rebuilds the rest
// from the history rows, so river_job stays small at high volume.
type RunArgs struct {
	PlaybookHistoryID uuid.UUID `json:"playbook_history_id"`
}

func (RunArgs) Kind() string { return "playbook_run" }

func (RunArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: queueName, MaxAttempts: maxAttempts}
}

// RunLoader rebuilds the message of a queued run. playbooks.RunLoader implements it.
type RunLoader interface {
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
	loader   RunLoader
	executor RunExecutor
}

func NewRunWorker(log logger.Logger, loader RunLoader, executor RunExecutor) *RunWorker {
	return &RunWorker{logger: log, loader: loader, executor: executor}
}

func (w *RunWorker) Work(ctx context.Context, job *river.Job[RunArgs]) error {
	id := job.Args.PlaybookHistoryID
	msg, err := w.loader.LoadRunMessage(ctx, id)
	if err == nil {
		err = w.executor.Run(ctx, msg)
		if !errors.Is(err, execution.ErrNotStarted) {
			// A failed run is already persisted by the executor, so it is not a
			// job error: returning it would make River retry and run the
			// connectors again.
			if err != nil {
				w.logger.Errorw("playbook run failed", "playbook_history_id", id, "error", err)
			}
			return nil
		}
	}

	// The run could not be built or started, and nothing ran: River retries
	// with backoff, and on the last attempt the run is failed, or it would
	// stay queued forever.
	if job.Attempt < job.MaxAttempts {
		return err
	}
	w.logger.Errorw("could not start the run, failing it", "playbook_history_id", id, "error", err)
	return w.executor.Abort(ctx, id, err)
}

// Enqueuer queues runs. It implements contracts.RunEnqueuer.
type Enqueuer struct {
	client *river.Client[pgx.Tx]
}

func NewEnqueuer(client *river.Client[pgx.Tx]) *Enqueuer { return &Enqueuer{client: client} }

// EnqueueRun joins the transaction in ctx when there is one.
func (e *Enqueuer) EnqueueRun(ctx context.Context, playbookHistoryID uuid.UUID) error {
	args := RunArgs{PlaybookHistoryID: playbookHistoryID}
	var err error
	if tx, ok := repository.TxFromContext(ctx); ok {
		_, err = e.client.InsertTx(ctx, tx, args, nil)
	} else {
		_, err = e.client.Insert(ctx, args, nil)
	}
	return err
}

// NewInsertClient is what the API uses: it queues jobs and works none.
func NewInsertClient(pool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{})
}

// NewWorkerClient works playbook runs, `concurrency` at a time. runTimeout caps
// one whole run.
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

// ModuleEventLogger implements contracts.ModuleEventPublisher. Nothing reacts
// to alert and incident events yet, so it only logs them. Event-triggered
// playbooks (on_create, on_update) will queue their runs from here.
type ModuleEventLogger struct{ logger logger.Logger }

func NewModuleEventLogger(log logger.Logger) *ModuleEventLogger {
	return &ModuleEventLogger{logger: log}
}

func (l *ModuleEventLogger) Publish(module, event string, _ any) error {
	l.logger.Debugw("module event", "module", module, "event", event)
	return nil
}
