package main

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("hello")

	conn, err := amqp.Dial("amqp://admin:changeme@localhost:5672")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	errs := ch.QueueBind("telemetry_queue",
		"telemetry_topic",
		"telemetry.ticks", false, nil)

	failOnError(errs, "Failed to bind to queue")

	msgs, err := ch.Consume("telemetry_queue", "", false, false, false, false, nil)
	failOnError(err, "Failed to consume queue")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf(" [x] %s", d.Body)
		}
	}()

	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
