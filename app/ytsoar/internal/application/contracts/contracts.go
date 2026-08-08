package contracts

import (
	"context"

	"github.com/google/uuid"
)

//go:generate mockgen -destination=mocks/task_publisher_mock.go -package=mocks . TaskPublisher

// TaskPublisher publishes a triggered playbook message to the message queue.
type TaskPublisher interface {
	SendMessage(message any) error
}

//go:generate mockgen -destination=mocks/status_broadcaster_mock.go -package=mocks . StatusBroadcaster

// StatusBroadcaster pushes playbook/task status updates to connected clients.
type StatusBroadcaster interface {
	Broadcast(data any)
}

//go:generate mockgen -destination=mocks/module_event_publisher_mock.go -package=mocks . ModuleEventPublisher

// ModuleEventPublisher announces an entity lifecycle event (alert.created,
// incident.updated, …) so consumers can react without the alerts and incidents
// services knowing they exist.
//
// Callers must publish only after their transaction commits: a publish inside
// WithinTransaction announces state that a rollback then discards, and there is
// no un-publish.
type ModuleEventPublisher interface {
	Publish(module, event string, entity any) error
}

//go:generate mockgen -destination=mocks/tx_manager_mock.go -package=mocks . TxManager

// TxManager runs fn inside a database transaction carried in the context, so
// repositories join it transparently and services never touch the driver.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

//go:generate mockgen -destination=mocks/user_directory_mock.go -package=mocks . UserDirectory

// UserDirectory resolves user ids to usernames for display inside timeline
// events. Without it an assignment event reads "assignee changed from 3f2a… to
// 9b1c…", and who it was assigned to is the entire content of that event.
type UserDirectory interface {
	UsernamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}
