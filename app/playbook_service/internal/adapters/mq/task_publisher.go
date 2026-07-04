package mq

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/yuudev14/ytsoar/internal/logging"
)

// TaskPublisher publishes triggered playbook messages to the playbook queue.
// It implements contracts.TaskPublisher.
type TaskPublisher struct {
	channel   *amqp.Channel
	queueName string
}

func NewTaskPublisher(conn *Connection, queueName string) (*TaskPublisher, error) {
	if _, err := conn.DeclareQueue(queueName); err != nil {
		return nil, err
	}
	return &TaskPublisher{
		channel:   conn.Channel,
		queueName: queueName,
	}, nil
}

func (p *TaskPublisher) SendMessage(message interface{}) error {
	jsonData, jsonErr := json.Marshal(message)
	if jsonErr != nil {
		return jsonErr
	}

	err := p.channel.PublishWithContext(
		context.Background(),
		"",          // exchange
		p.queueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         jsonData,
		})
	if err != nil {
		return err
	}

	logging.Sugar.Infow("successfully pushed the message", "jsonData", string(jsonData))
	return nil
}
