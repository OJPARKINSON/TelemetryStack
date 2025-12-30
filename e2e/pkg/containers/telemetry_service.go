package containers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

func StartTelemetryService(T *testing.T, ctx context.Context) *testcontainers.DockerContainer {
	df := testcontainers.FromDockerfile{
		Context:    filepath.Join(".", "../telemetryService/golang/Dockerfile"),
		Dockerfile: "Dockerfile",
		Repo:       "IRacingService",
		Tag:        "latest",
		BuildArgs:  map[string]*string{},
	}

	container, err := testcontainers.Run(
		ctx,
		"questdb/questdb:latest",
		testcontainers.WithDockerfile(df),
		testcontainers.WithExposedPorts("9092:9092"),
		testcontainers.WithEnv(map[string]string{
			"QUESTDB_URL":   "questdb:8812;username=admin;password=quest",
			"QUESTDB_HOST":  "questdb",
			"QUESTDB_PORT":  "9000",
			"RABBITMQ_HOST": "rabbitmq",
		}),
		testcontainers.WithWaitStrategy(
		// wait.ForHealthCheck(),
		// wait.ForListeningPort("8812/tcp"),
		// wait.ForLog("Ready to accept connections"),
		),
	)
	if err != nil {
		fmt.Println("Error running questDB server, ", err)
	}

	return container
}
