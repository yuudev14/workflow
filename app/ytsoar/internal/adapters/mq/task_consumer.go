package mq

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// TaskConsumer feeds triggered playbooks from the queue into the executor.
// prefetch is how many playbooks one worker runs concurrently: each delivery
// runs in its own goroutine and the MQ Qos caps the unacked deliveries in
// flight. A delivery is acked after the run finishes either way, because
// failures are persisted and an executed playbook must never be requeued.
type TaskConsumer struct {
	logger   logger.Logger
	channel  *amqp.Channel
	queue    string
	executor *execution.Executor
	prefetch int
}

func NewTaskConsumer(log logger.Logger, conn *Connection, queue string, executor *execution.Executor, prefetch int) (*TaskConsumer, error) {
	if _, err := conn.DeclareQueue(queue); err != nil {
		return nil, err
	}
	if prefetch < 1 {
		prefetch = 1
	}
	return &TaskConsumer{
		logger:   log,
		channel:  conn.Channel,
		queue:    queue,
		executor: executor,
		prefetch: prefetch,
	}, nil
}

func (c *TaskConsumer) Start(ctx context.Context) error {
	if err := c.channel.Qos(c.prefetch, 0, false); err != nil {
		return err
	}
	messages, err := c.channel.Consume(
		c.queue,
		"",
		false, // manual ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}

	c.logger.Infow("task consumer started", "queue", c.queue, "prefetch", c.prefetch)
	var wg sync.WaitGroup
	defer wg.Wait() // in-flight runs still persist their status on shutdown
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-messages:
			if !ok {
				return errors.New("mq channel closed")
			}
			wg.Go(func() {
				c.handle(ctx, delivery)
			})
		}
	}
}

func (c *TaskConsumer) handle(ctx context.Context, delivery amqp.Delivery) {
	var msg domain.TaskMessage
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		c.logger.Errorw("undecodable task message", "error", err)
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			c.logger.Errorw("nack failed", "error", nackErr)
		}
		return
	}
	if err := c.executor.Run(ctx, msg); err != nil {
		c.logger.Errorw("playbook run failed",
			"playbook_history_id", msg.PlaybookHistoryId, "error", err)
	}
	if err := delivery.Ack(false); err != nil {
		c.logger.Errorw("ack failed", "error", err)
	}
}
