package mq

import (
	"context"
	"encoding/json"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuudev14/ytsoar/internal/application/contracts"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// StatusConsumer runs in the API process: it binds a server-named exclusive
// queue to the status fanout exchange (so every replica receives every event)
// and forwards each event's data to the WS hub.
type StatusConsumer struct {
	logger   logger.Logger
	channel  *amqp.Channel
	exchange string
	hub      contracts.StatusBroadcaster
}

func NewStatusConsumer(log logger.Logger, conn *Connection, exchange string, hub contracts.StatusBroadcaster) *StatusConsumer {
	return &StatusConsumer{
		logger:   log,
		channel:  conn.Channel,
		exchange: exchange,
		hub:      hub,
	}
}

func (c *StatusConsumer) Start(ctx context.Context) error {
	if err := c.channel.ExchangeDeclare(c.exchange, "fanout", true, false, false, false, nil); err != nil {
		return err
	}
	queue, err := c.channel.QueueDeclare(
		"",    // server-named
		false, // durable
		true,  // auto-delete
		true,  // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return err
	}
	if err := c.channel.QueueBind(queue.Name, "", c.exchange, false, nil); err != nil {
		return err
	}
	messages, err := c.channel.Consume(
		queue.Name,
		"",
		true, // auto-ack: status events are fire-and-forget
		true, // exclusive
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	c.logger.Infow("status consumer bound", "exchange", c.exchange, "queue", queue.Name)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case delivery, ok := <-messages:
			if !ok {
				return errors.New("status channel closed")
			}
			var event struct {
				Event string          `json:"event"`
				Data  json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(delivery.Body, &event); err != nil {
				c.logger.Errorw("undecodable status event", "error", err)
				continue
			}
			c.hub.Broadcast(event.Data)
		}
	}
}
