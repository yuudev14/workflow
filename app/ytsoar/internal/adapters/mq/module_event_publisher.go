package mq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuudev14/ytsoar/internal/logger"
)

// ModuleEventPublisher announces entity lifecycle events on a topic exchange so
// a consumer can bind to a slice of them (`alert.#`, `*.created`) rather than
// receiving every event like the fanout status exchange. It implements
// contracts.ModuleEventPublisher.
type ModuleEventPublisher struct {
	logger   logger.Logger
	channel  *amqp.Channel
	exchange string
}

func NewModuleEventPublisher(log logger.Logger, conn *Connection, exchange string) (*ModuleEventPublisher, error) {
	if err := conn.Channel.ExchangeDeclare(
		exchange,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return nil, err
	}
	return &ModuleEventPublisher{
		logger:   log,
		channel:  conn.Channel,
		exchange: exchange,
	}, nil
}

func (p *ModuleEventPublisher) Publish(module, event string, entity any) error {
	body, err := json.Marshal(map[string]any{
		"module": module,
		"event":  event,
		"data":   entity,
	})
	if err != nil {
		return err
	}

	return p.channel.PublishWithContext(
		context.Background(),
		p.exchange,
		module+"."+event,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}
