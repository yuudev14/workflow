package mq

import (
	"github.com/streadway/amqp"
	"github.com/yuudev14-workflow/workflow-service/environment"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
)

var (
	MQConn        *amqp.Connection
	MQChannel     *amqp.Channel
	SenderQueue   amqp.Queue
	ReceiverQueue amqp.Queue
)

func ConnectToMQ() {
	logging.Sugar.Infof("connecting to message queue %v...", environment.Settings.MQ_URL)
	conn, err := amqp.Dial(environment.Settings.MQ_URL)
	if err != nil {
		logging.Sugar.Panicf("%v", err)
	}
	logging.Sugar.Info("connected to message queue...")

	MQConn = conn

	ch, chErr := MQConn.Channel()
	logging.Sugar.Info("open channel in message queue...")

	if chErr != nil {
		logging.Sugar.Panicf("%v", chErr)
	}

	MQChannel = ch
	declareQueues(MQChannel)

}

func declareQueues(ch *amqp.Channel) {
	logging.Sugar.Info("Declaring queues")
	declareSenderQueue(ch)
	declareReceiverQueue(ch)
}

func declareSenderQueue(ch *amqp.Channel) {
	logging.Sugar.Info("Declaring sender queue")
	// Declare a queue
	q, err := ch.QueueDeclare(
		environment.Settings.SenderQueueName, // name
		true,                                 // durable
		false,                                // delete when unused
		false,                                // exclusive
		false,                                // no-wait
		nil,                                  // arguments
	)
	if err != nil {
		logging.Sugar.Panicf("%v", err)
	}
	SenderQueue = q
}

func declareReceiverQueue(ch *amqp.Channel) {
	logging.Sugar.Info("Declaring receiver queue")
	// Declare a queue
	q, err := ch.QueueDeclare(
		environment.Settings.ReceiverQueueName, // name
		true,                                   // durable
		false,                                  // delete when unused
		false,                                  // exclusive
		false,                                  // no-wait
		nil,                                    // arguments
	)
	if err != nil {
		logging.Sugar.Panicf("%v", err)
	}
	ReceiverQueue = q
}
