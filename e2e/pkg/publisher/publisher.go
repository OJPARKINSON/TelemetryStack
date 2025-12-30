package publisher

import (
	"context"
	"fmt"

	"github.com/ojparkinson/telemetryService/internal/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"google.golang.org/protobuf/proto"
)

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(rabbitMQ string) (*Publisher, error) {
	conn, err := amqp.Dial(rabbitMQ)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Publisher{
		conn:    conn,
		channel: channel,
	}, nil
}

func (p *Publisher) PublishBatch(rabbitmq *rabbitmq.RabbitMQContainer, batch *messaging.TelemetryBatch, ctx context.Context) {
	_, err := proto.Marshal(batch)
	if err != nil {
		fmt.Println("marshal ")
	}

	//
}
