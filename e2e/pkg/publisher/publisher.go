package publisher

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/protobuf/proto"
)

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(rabbitmq *testcontainers.DockerContainer, ctx context.Context) (*Publisher, error) {
	host, _ := rabbitmq.Host(ctx)
	port, _ := rabbitmq.MappedPort(ctx, "5672")

	conn, err := amqp.Dial(fmt.Sprintf("amqp://admin:changeme@%s:%s", host, port.Port()))
	if err != nil {
		return nil, err
	}

	fmt.Println("Connecting channel")
	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Publisher{
		conn:    conn,
		channel: channel,
	}, nil
}

func (p *Publisher) PublishBatch(rabbitmq *testcontainers.DockerContainer, batches []*TelemetryBatch, ctx context.Context) {
	for _, batch := range batches {

		data, err := proto.Marshal(batch)
		if err != nil {
			fmt.Println("marshal ")
		}

		err2 := p.channel.PublishWithContext(ctx, "telemetry_topic", "telemetry.ticks", false, false,
			amqp.Publishing{
				ContentType:  "application/x-protobuf",
				Body:         data, // marshaled protobuf
				DeliveryMode: amqp.Transient,
				Timestamp:    time.Now(),
				MessageId:    batch.BatchId,
			})

		if err2 != nil {
			fmt.Println(err2)
		}
	}
}
