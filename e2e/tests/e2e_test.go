package tests

import (
	"context"
	"testing"

	"github.com/ojparkinson/IRacing-Display/e2e/pkg/containers"
	"github.com/ojparkinson/IRacing-Display/e2e/pkg/publisher"
)

func TestBasicE2E(t *testing.T) {
	ctx := context.Background()

	questdb := containers.SpinUpQuestDB(t, ctx)
	rabbitmq := containers.StartRabbitMQ(t, ctx)
	telemetryService := containers.StartTelemetryService(t, ctx)

	batches := publisher.GenerateBatches(1)

	publisher.Publish(rabbitmq, batches, ctx)

	t.Cleanup(func() {
		questdb.Terminate(ctx)
		rabbitmq.Terminate(ctx)
		telemetryService.Terminate(ctx)
	})
}
