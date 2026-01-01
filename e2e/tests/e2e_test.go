package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ojparkinson/IRacing-Display/e2e/pkg/containers"
	"github.com/ojparkinson/IRacing-Display/e2e/pkg/publisher"
	"github.com/ojparkinson/IRacing-Display/e2e/pkg/verification"
)

func TestBasicE2E(t *testing.T) {
	ctx := context.Background()

	network, err := containers.CreateNetwork(ctx)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	containers.SpinUpQuestDB(t, ctx, network)
	rabbitmqC := containers.StartRabbitMQ(t, ctx, network)
	containers.StartTelemetryService(t, ctx, network)

	host, _ := rabbitmqC.Host(ctx)
	port, _ := rabbitmqC.MappedPort(ctx, "5672")

	batches := publisher.GenerateBatch(300, 10000)

	rabbitmq, err := publisher.NewPublisher(fmt.Sprintf("amqp://admin:changeme@%s:%s", host, port.Port()))
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}

	rabbitmq.PublishBatch(rabbitmqC, batches, ctx)

	time.Sleep(2 * time.Minute)

	verification.RecordsStored()

	// Cleanup is handled automatically by t.Cleanup() in each container function
}
