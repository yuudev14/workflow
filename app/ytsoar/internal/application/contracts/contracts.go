package contracts

import "context"

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

//go:generate mockgen -destination=mocks/tx_manager_mock.go -package=mocks . TxManager

// TxManager runs fn inside a database transaction carried in the context, so
// repositories join it transparently and services never touch the driver.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
