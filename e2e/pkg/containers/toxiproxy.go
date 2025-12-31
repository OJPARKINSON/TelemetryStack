package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SpinUpToxiProxy(t *testing.T, ctx context.Context) *testcontainers.DockerContainer {
	container, err := testcontainers.Run(
		ctx,
		"ghcr.io/shopify/toxiproxy:latest",
		testcontainers.WithExposedPorts("8474:8474"),
		testcontainers.WithEnv(map[string]string{}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("8474"),
			// wait.ForLog("Ready to accept connections"),
		),
	)
	if err != nil {
		fmt.Println("Error running questDB server, ", err)
	}

	return container
}
