package queue

import (
	"fmt"
	"log"

	"github.com/ojparkinson/telemetryService/internal/config"
	"github.com/ojparkinson/telemetryService/internal/messaging"
	"github.com/ojparkinson/telemetryService/internal/persistance"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
)

type Subscriber struct {
}

func NewSubscriber() *Subscriber {
	return &Subscriber{}
}

func (m *Subscriber) Subscribe(config *config.Config) {
	conn, err := amqp.Dial("amqp://admin:changeme@" + config.RabbitMQHost + ":5672")
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

	for event := range msgs {
		batch := &messaging.TelemetryBatch{}
		err := proto.Unmarshal(event.Body, batch)
		if err != nil {
			fmt.Println("error unmarshalling: ", err)
		}
		fmt.Printf("data: %v \n", batch.SessionId)

		go parseBatch(batch, event)
	}
}

func parseBatch(batch *messaging.TelemetryBatch, event amqp.Delivery) {
	for _, tick := range batch.Records {
		fmt.Printf("data: %v \n", tick.Speed)
		persistance.SaveTickToDB(tick)
	}

	event.Ack(true)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
