package containers

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartTelemetryService(t *testing.T, ctx context.Context, nw *testcontainers.DockerNetwork) *testcontainers.DockerContainer {
	df := testcontainers.FromDockerfile{
		Context:    filepath.Join("..", "..", "telemetryService", "golang"),
		Dockerfile: "Dockerfile",
		Repo:       "IRacingService",
		Tag:        "latest",
		BuildArgs:  map[string]*string{},
	}

	container, err := testcontainers.Run(
		ctx,
		"",
		testcontainers.WithDockerfile(df),
		testcontainers.WithExposedPorts("9092:9092"),
		testcontainers.WithEnv(map[string]string{
			"QUESTDB_URL":      "questdb:8812;username=admin;password=quest",
			"QUESTDB_HOST":     "questdb",
			"QUESTDB_PORT":     "9000",
			"RABBITMQ_HOST":    "rabbitmq",
			"SENDER_POOL_SIZE": "10",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Starting to consume messages from RabbitMQ"),
		),
		network.WithNetwork([]string{"telemetry-service"}, nw),
	)
	if err != nil {
		t.Fatalf("Error running telemetry service: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate telemetry service container: %v", err)
		}
	})

	return container
}
