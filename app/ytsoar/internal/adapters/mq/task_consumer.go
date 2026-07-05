package mq

import (
	"context"
	"encoding/json"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// TaskConsumer feeds triggered playbooks from the queue into the executor.
// prefetch=1 keeps one playbook per worker; the delivery is acked after the
// run finishes either way, because failures are persisted and an executed
// playbook must never be requeued.
type TaskConsumer struct {
	logger   logger.Logger
	channel  *amqp.Channel
	queue    string
	executor *execution.Executor
}

func NewTaskConsumer(log logger.Logger, conn *Connection, queue string, executor *execution.Executor) (*TaskConsumer, error) {
	if _, err := conn.DeclareQueue(queue); err != nil {
		return nil, err
	}
	return &TaskConsumer{
		logger:   log,
		channel:  conn.Channel,
		queue:    queue,
		executor: executor,
	}, nil
}

func (c *TaskConsumer) Start(ctx context.Context) error {
	if err := c.channel.Qos(1, 0, false); err != nil {
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

	c.logger.Infow("task consumer started", "queue", c.queue)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-messages:
			if !ok {
				return errors.New("mq channel closed")
			}
			var msg domain.TaskMessage
			if err := json.Unmarshal(delivery.Body, &msg); err != nil {
				c.logger.Errorw("undecodable task message", "error", err)
				if nackErr := delivery.Nack(false, false); nackErr != nil {
					c.logger.Errorw("nack failed", "error", nackErr)
				}
				continue
			}
			if err := c.executor.Run(ctx, msg); err != nil {
				c.logger.Errorw("playbook run failed",
					"playbook_history_id", msg.PlaybookHistoryId, "error", err)
			}
			if err := delivery.Ack(false); err != nil {
				c.logger.Errorw("ack failed", "error", err)
			}
		}
	}
}
