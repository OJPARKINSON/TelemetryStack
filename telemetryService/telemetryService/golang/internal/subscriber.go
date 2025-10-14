package internal

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Subscriber struct {
}

func NewSubscriber() *Subscriber {
	return &Subscriber{}
}

func (m *Subscriber) Subscribe() {
	conn, err := amqp.Dial("amqp://admin:changeme@localhost:5672")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	queue, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer queue.Close()

	errs := queue.QueueBind("telemetry_queue",
		"telemetry.ticks",
		"telemetry_topic", false, nil)

	failOnError(errs, "Failed to bind to queue")

	msgs, err := queue.Consume("telemetry_queue", "", false, false, false, false, nil)
	failOnError(err, "Failed to consume queue")

	fmt.Println("Starting to consume")
	for event := range msgs {
		fmt.Printf("data: %v \n", string(event.Body))
		event.Ack(true)
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
