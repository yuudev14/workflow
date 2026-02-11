package mq

import (
	"github.com/streadway/amqp"
	"github.com/yuudev14-workflow/workflow-service/internal/infra/logging"
)

type MQStruct struct {
	MQConn        *amqp.Connection
	MQChannel     *amqp.Channel
	SenderQueue   amqp.Queue
	ReceiverQueue amqp.Queue
}

func ConnectToMQ(mqURL string, senderQueueName string, receiverQueueName string) MQStruct {
	logging.Sugar.Infof("connecting to message queue %v...", mqURL)
	conn, err := amqp.Dial(mqURL)
	if err != nil {
		logging.Sugar.Panicf("%v", err)
	}
	logging.Sugar.Info("connected to message queue...")

	MQConn := conn

	ch, chErr := MQConn.Channel()
	logging.Sugar.Info("open channel in message queue...")

	if chErr != nil {
		logging.Sugar.Panicf("%v", chErr)
	}

	MQChannel := ch

	senderQueue, receiverQueue := declareQueues(MQChannel, senderQueueName, receiverQueueName)
	return MQStruct{
		MQConn:        MQConn,
		MQChannel:     MQChannel,
		SenderQueue:   senderQueue,
		ReceiverQueue: receiverQueue,
	}

}

func declareQueues(ch *amqp.Channel, senderQueueName string, receiverQueueName string) (amqp.Queue, amqp.Queue) {
	logging.Sugar.Info("Declaring queues")
	senderQueue := declareSenderQueue(ch, senderQueueName)
	receiverQueue := declareReceiverQueue(ch, receiverQueueName)
	return senderQueue, receiverQueue
}

func declareSenderQueue(ch *amqp.Channel, queueName string) amqp.Queue {
	logging.Sugar.Info("Declaring sender queue")
	// Declare a queue
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		logging.Sugar.Panic("%v", err)
	}
	return q
}

func declareReceiverQueue(ch *amqp.Channel, queueName string) amqp.Queue {
	logging.Sugar.Info("Declaring receiver queue")
	// Declare a queue
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		logging.Sugar.Panicf("%v", err)
	}
	return q
}
