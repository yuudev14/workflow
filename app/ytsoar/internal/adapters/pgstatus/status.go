// Package pgstatus carries playbook and task status events from the worker to
// every API replica over Postgres LISTEN/NOTIFY, so no message broker is needed.
// Events are fire and forget, like the WS push they feed: a client that missed
// one catches up on its next fetch.
package pgstatus

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

const channel = "ytsoar_status"

// event is the NOTIFY payload. It carries the row id only: a NOTIFY payload is
// capped at 8000 bytes and a task result can be larger.
type event struct {
	Event string    `json:"event"`
	ID    uuid.UUID `json:"id"`
}

// Notifier implements execution.StatusPublisher.
type Notifier struct{ pool *pgxpool.Pool }

func NewNotifier(pool *pgxpool.Pool) *Notifier { return &Notifier{pool: pool} }

func (n *Notifier) Publish(ctx context.Context, name string, id uuid.UUID) error {
	body, err := json.Marshal(event{Event: name, ID: id})
	if err != nil {
		return err
	}
	_, err = n.pool.Exec(ctx, "SELECT pg_notify($1, $2)", channel, string(body))
	return err
}

// PlaybookReader and TaskReader load the rows an event points at.
type PlaybookReader interface {
	GetPlaybookHistoryById(ctx context.Context, id uuid.UUID) (*domain.PlaybookHistoryResponse, error)
}

type TaskReader interface {
	GetTaskHistoryById(ctx context.Context, id uuid.UUID) (*domain.TaskHistory, error)
}

// Listener runs in the API: it loads the row of each event and forwards it to
// the WS hub, in the {event, data} envelope the clients route on.
type Listener struct {
	logger    logger.Logger
	pool      *pgxpool.Pool
	playbooks PlaybookReader
	tasks     TaskReader
	hub       contracts.StatusBroadcaster
}

func NewListener(log logger.Logger, pool *pgxpool.Pool, playbooks PlaybookReader, tasks TaskReader, hub contracts.StatusBroadcaster) *Listener {
	return &Listener{logger: log, pool: pool, playbooks: playbooks, tasks: tasks, hub: hub}
}

// Start listens until ctx ends, reconnecting with a growing delay when the
// connection drops.
func (l *Listener) Start(ctx context.Context) error {
	delay := time.Second
	for ctx.Err() == nil {
		if err := l.listen(ctx, func() { delay = time.Second }); err != nil && ctx.Err() == nil {
			l.logger.Errorw("status listener lost its connection", "error", err, "retry_in", delay)
		}
		select {
		case <-ctx.Done():
		case <-time.After(delay):
			delay = min(delay*2, 30*time.Second)
		}
	}
	return ctx.Err()
}

func (l *Listener) listen(ctx context.Context, onListening func()) error {
	// A dedicated connection, not one from the pool: it is held for the life of
	// the process, and a pool with a held connection never finishes closing.
	conn, err := pgx.ConnectConfig(ctx, l.pool.Config().ConnConfig)
	if err != nil {
		return err
	}
	defer conn.Close(context.WithoutCancel(ctx))
	if _, err := conn.Exec(ctx, "LISTEN "+channel); err != nil {
		return err
	}
	l.logger.Infow("status listener started", "channel", channel)
	onListening()

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		l.forward(ctx, notification.Payload)
	}
}

func (l *Listener) forward(ctx context.Context, payload string) {
	var e event
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		l.logger.Errorw("undecodable status event", "error", err)
		return
	}
	var (
		data any
		err  error
	)
	switch e.Event {
	case "playbook_status":
		data, err = l.playbooks.GetPlaybookHistoryById(ctx, e.ID)
	case "task_status":
		data, err = l.tasks.GetTaskHistoryById(ctx, e.ID)
	default:
		l.logger.Warnw("unknown status event", "event", e.Event)
		return
	}
	if err != nil {
		l.logger.Errorw("could not load the row of a status event", "event", e.Event, "id", e.ID, "error", err)
		return
	}
	l.hub.Broadcast(map[string]any{"event": e.Event, "data": data})
}
