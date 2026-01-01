package containers

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SpinUpToxiProxy(t *testing.T, ctx context.Context, nw *testcontainers.DockerNetwork) *testcontainers.DockerContainer {
	container, err := testcontainers.Run(
		ctx,
		"ghcr.io/shopify/toxiproxy:latest",
		testcontainers.WithExposedPorts("8474:8474"),
		testcontainers.WithEnv(map[string]string{}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("8474"),
		),
		network.WithNetwork([]string{"toxiproxy"}, nw),
	)
	if err != nil {
		t.Fatalf("Error running Toxiproxy: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate Toxiproxy container: %v", err)
		}
	})

	return container
}
