// Package jobs runs playbook runs on River (a job queue in Postgres). The API queues a run
// in the same transaction as its history rows, so a run is never lost and never queued
// without its history.
package jobs

import (
	"context"
	"errors"
	"fmt"
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
	// eventQueue has its own workers, so a long run does not hold back the events
	eventQueue = "playbook_event"
	// maxAttempts is 2. A failed run is not retried and a started run never starts again.
	// Attempt 2 is for a job that River takes back (worker died, final status not saved, or
	// Abort failed). If the run started, attempt 2 only marks it failed. If it never started
	// (Abort failed), attempt 2 can start it. A run that cannot start is not retried: the
	// worker marks it failed at once.
	maxAttempts = 2
	// rescueGrace: time after the run timeout when River decides the worker died. It must be
	// longer than the run timeout, or River takes a live run.
	rescueGrace = 5 * time.Minute
)

// RunArgs is the job payload: only the run ID. The worker builds the rest from the history
// rows, so river_job stays small.
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
		switch {
		case errors.Is(err, execution.ErrNotStarted):
			// the run could not start, go to Abort below
		case errors.Is(err, execution.ErrNotFinished):
			// the final status was not saved. Return the error so River retries, and attempt 2 marks
			// the run failed.
			return err
		default:
			// a failed run is not a job error. The executor already saved it, and a retry would run
			// the connectors again.
			if err != nil {
				w.logger.Errorw("playbook run failed", "playbook_history_id", id, "error", err)
			}
			return nil
		}
	}

	// the worker is stopping (or the job timed out). Return the error without failing the run.
	// River requeues it without using an attempt.
	if ctx.Err() != nil {
		return err
	}

	// the run could not be built or started. Do not retry: mark it failed so the user sees the
	// error. If Abort fails (database down), return the error and attempt 2 tries again. On
	// attempt 2 the playbook may have run already (ErrNotFinished), so the text says so.
	cause := err
	if job.Attempt > 1 {
		cause = fmt.Errorf("an earlier attempt did not finish this run: %w", err)
	}
	w.logger.Errorw("could not start the run, failing it", "playbook_history_id", id, "attempt", job.Attempt, "error", err)
	return w.executor.Abort(ctx, id, cause)
}

// RunForEventArgs is an alert or incident event. The worker finds the playbooks it
// triggers.
type RunForEventArgs struct {
	ModuleType string    `json:"module_type"`
	Event      string    `json:"event"`
	RecordID   uuid.UUID `json:"record_id"`
}

func (RunForEventArgs) Kind() string { return "playbook_run_for_event" }

// a retry is safe: all runs of one event are queued in one transaction
func (RunForEventArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: eventQueue, MaxAttempts: 3}
}

// EventRunner queues the runs of an event. playbooks.PlaybookApplicationServiceImpl
// implements it.
type EventRunner interface {
	RunForEvent(ctx context.Context, moduleType string, event string, recordID uuid.UUID, occurredAt time.Time) error
}

type RunForEventWorker struct {
	river.WorkerDefaults[RunForEventArgs]
	logger logger.Logger
	runner EventRunner
}

func NewRunForEventWorker(log logger.Logger, runner EventRunner) *RunForEventWorker {
	return &RunForEventWorker{logger: log, runner: runner}
}

func (w *RunForEventWorker) Work(ctx context.Context, job *river.Job[RunForEventArgs]) error {
	a := job.Args
	// CreatedAt is the time of the change: River sets it in the transaction of the change
	err := w.runner.RunForEvent(ctx, a.ModuleType, a.Event, a.RecordID, job.CreatedAt)
	if err != nil {
		w.logger.Errorw("could not queue the runs of an event", "module", a.ModuleType, "event", a.Event, "record_id", a.RecordID, "error", err)
	}
	return err
}

// Enqueuer queues runs and module events. It implements contracts.RunEnqueuer and
// contracts.ModuleEventPublisher.
type Enqueuer struct {
	client *river.Client[pgx.Tx]
	// txFrom finds the transaction in ctx. main.go injects it, so this adapter does not
	// import the repository adapter.
	txFrom func(context.Context) (pgx.Tx, bool)
}

// NewEnqueuer is for the API. Its River client only inserts jobs, the worker process runs
// them.
func NewEnqueuer(pool *pgxpool.Pool, txFrom func(context.Context) (pgx.Tx, bool)) (*Enqueuer, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return nil, err
	}
	return &Enqueuer{client: client, txFrom: txFrom}, nil
}

// EnqueueRun joins the transaction in ctx, if ctx has one.
func (e *Enqueuer) EnqueueRun(ctx context.Context, playbookHistoryID uuid.UUID) error {
	return e.insert(ctx, RunArgs{PlaybookHistoryID: playbookHistoryID})
}

// Publish implements contracts.ModuleEventPublisher. It joins the transaction in ctx, if
// ctx has one.
func (e *Enqueuer) Publish(ctx context.Context, module, event string, recordID uuid.UUID) error {
	return e.insert(ctx, RunForEventArgs{ModuleType: module, Event: event, RecordID: recordID})
}

func (e *Enqueuer) insert(ctx context.Context, args river.JobArgs) error {
	var err error
	if tx, ok := e.txFrom(ctx); ok {
		_, err = e.client.InsertTx(ctx, tx, args, nil)
	} else {
		_, err = e.client.Insert(ctx, args, nil)
	}
	return err
}

// NewWorkerClient runs `concurrency` runs at the same time. runTimeout limits one run.
// When the worker stops, River waits stopGrace, then cancels the runs. A cancelled run is
// saved as failed.
func NewWorkerClient(pool *pgxpool.Pool, worker *RunWorker, eventWorker *RunForEventWorker, concurrency int, runTimeout, stopGrace time.Duration) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	river.AddWorker(workers, eventWorker)
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			queueName:  {MaxWorkers: max(concurrency, 1)},
			eventQueue: {MaxWorkers: 1},
		},
		Workers:              workers,
		JobTimeout:           runTimeout,
		RescueStuckJobsAfter: runTimeout + rescueGrace,
		SoftStopTimeout:      stopGrace,
	})
}
