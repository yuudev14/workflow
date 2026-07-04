package contracts

//go:generate mockgen -destination=mocks/task_publisher_mock.go -package=mocks . TaskPublisher

// TaskPublisher publishes a triggered playbook message to the message queue.
type TaskPublisher interface {
	SendMessage(message interface{}) error
}

//go:generate mockgen -destination=mocks/status_broadcaster_mock.go -package=mocks . StatusBroadcaster

// StatusBroadcaster pushes playbook/task status updates to connected clients.
type StatusBroadcaster interface {
	Broadcast(data interface{})
}
