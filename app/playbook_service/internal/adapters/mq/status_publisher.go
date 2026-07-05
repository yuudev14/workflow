package mq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// StatusPublisher fans playbook/task status events out of the worker so every
// API replica can forward them to its WS hub. It implements
// execution.StatusPublisher.
type StatusPublisher struct {
	logger   logger.Logger
	channel  *amqp.Channel
	exchange string
}

func NewStatusPublisher(log logger.Logger, conn *Connection, exchange string) (*StatusPublisher, error) {
	if err := conn.Channel.ExchangeDeclare(
		exchange,
		"fanout",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return nil, err
	}
	return &StatusPublisher{
		logger:   log,
		channel:  conn.Channel,
		exchange: exchange,
	}, nil
}

func (p *StatusPublisher) Publish(event string, data interface{}) error {
	body, err := json.Marshal(map[string]interface{}{
		"event": event,
		"data":  data,
	})
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(
		context.Background(),
		p.exchange,
		"",    // routing key ignored by fanout
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}
