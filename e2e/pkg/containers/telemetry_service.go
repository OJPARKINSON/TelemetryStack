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
	const name = "e2e-telemetryService"
	removeContainerByName(ctx, name)

	df := testcontainers.FromDockerfile{
		Context:    filepath.Join("..", "..", "telemetryService", "golang"),
		Dockerfile: "Dockerfile",
		Repo:       "IRacingService",
		Tag:        "latest",
		KeepImage:  true,
		BuildArgs:  map[string]*string{},
	}

	container, err := testcontainers.Run(
		ctx,
		"",
		testcontainers.WithDockerfile(df),
		testcontainers.WithExposedPorts("9092/tcp", "6060/tcp", "8010/tcp"),
		testcontainers.WithName(name),
		testcontainers.WithEnv(map[string]string{
			"QUESTDB_URL":      "questdb:8812;username=admin;password=quest",
			"QUESTDB_HOST":     "questdb",
			"QUESTDB_PORT":     "9000",
			"SENDER_POOL_SIZE": "60",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Starting to consume tick batches"),
		),
		network.WithNetwork([]string{"telemetry-service"}, nw),
	)
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("Error running telemetry service: %v", err)
	}

	return container
}
