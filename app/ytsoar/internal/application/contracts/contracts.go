package contracts

import (
	"context"

	"github.com/google/uuid"
	"github.com/yuudev14/ytsoar/internal/domain"
)

//go:generate mockgen -destination=mocks/audit_writer_mock.go -package=mocks . AuditWriter

// AuditWriter writes one audit log row. Each service that audits takes this one port.
type AuditWriter interface {
	Insert(ctx context.Context, entry domain.AuditEntry) error
}

//go:generate mockgen -destination=mocks/run_enqueuer_mock.go -package=mocks . RunEnqueuer

// RunEnqueuer queues a playbook run for a worker. It joins the transaction in ctx, so the
// job and the history rows commit or roll back together. A run is never queued without its
// history.
type RunEnqueuer interface {
	EnqueueRun(ctx context.Context, playbookHistoryID uuid.UUID) error
}

//go:generate mockgen -destination=mocks/status_broadcaster_mock.go -package=mocks . StatusBroadcaster

// StatusBroadcaster sends playbook and task status updates to the connected clients.
type StatusBroadcaster interface {
	Broadcast(data any)
}

//go:generate mockgen -destination=mocks/module_event_publisher_mock.go -package=mocks . ModuleEventPublisher

// ModuleEventPublisher announces an entity event (alert.created, incident.updated, ...),
// so consumers can react without the alerts and incidents services knowing them.
//
// Publish inside the transaction of the change. It joins the transaction in ctx, so a
// rollback also drops the event, and a commit never loses it.
type ModuleEventPublisher interface {
	Publish(ctx context.Context, module, event string, recordID uuid.UUID) error
}

//go:generate mockgen -destination=mocks/tx_manager_mock.go -package=mocks . TxManager

// TxManager runs fn in a database transaction carried in the context, so repositories join
// it and services never touch the driver.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

//go:generate mockgen -destination=mocks/user_directory_mock.go -package=mocks . UserDirectory

// UserDirectory turns user ids into usernames for timeline events. Without it, an
// assignment event reads "assignee changed from 3f2a… to 9b1c…", and who got the alert is
// the whole point of the event.
type UserDirectory interface {
	UsernamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}
