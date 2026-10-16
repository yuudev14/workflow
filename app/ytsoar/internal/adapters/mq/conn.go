package mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// Connection wraps an AMQP connection and channel pair.
type Connection struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func Connect(mqURL string) (*Connection, error) {
	conn, err := amqp.Dial(mqURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &Connection{
		Conn:    conn,
		Channel: ch,
	}, nil
}

func (c *Connection) Close() {
	c.Channel.Close()
	c.Conn.Close()
}
