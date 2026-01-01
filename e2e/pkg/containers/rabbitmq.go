package containers

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartRabbitMQ(t *testing.T, ctx context.Context, nw *testcontainers.DockerNetwork) *testcontainers.DockerContainer {
	// Get absolute path for Docker bind mount
	definitionsPath, err := filepath.Abs(filepath.Join("..", "..", "config", "definitions.json"))
	if err != nil {
		t.Fatalf("Failed to resolve absolute path for definitions.json: %v", err)
	}

	enabledPluginsPath, err := filepath.Abs(filepath.Join("..", "..", "config", "enabled_plugins"))
	if err != nil {
		t.Fatalf("Failed to resolve enabled_plugins path: %v", err)
	}

	rabbitmqConfPath, err := filepath.Abs(filepath.Join("..", "..", "config", "rabbitmq.conf"))

	container, err := testcontainers.Run(ctx, "rabbitmq:4.1",
		testcontainers.WithExposedPorts("5672/tcp", "15672/tcp", "15692/tcp"),
		testcontainers.WithMounts(
			testcontainers.BindMount(definitionsPath,
				testcontainers.ContainerMountTarget("/etc/rabbitmq/definitions.json")),
			testcontainers.BindMount(enabledPluginsPath,
				testcontainers.ContainerMountTarget("/etc/rabbitmq/enabled_plugins")),
			testcontainers.BindMount(rabbitmqConfPath,
				testcontainers.ContainerMountTarget("/etc/rabbitmq/rabbitmq.conf")),
		),
		testcontainers.WithEnv(map[string]string{
			"RABBITMQ_MANAGEMENT_LOAD_DEFINITIONS": "/etc/rabbitmq/definitions.json",
			"RABBITMQ_LOOPBACK_USERS":              "none",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForListeningPort("5672/tcp"),  // AMQP
				wait.ForListeningPort("15672/tcp"), // Management
			),
		),
		network.WithNetwork([]string{"rabbitmq"}, nw),
	)

	if err != nil {
		t.Fatalf("Error running RabbitMQ server: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate rabbitmq container: %v", err)
		}
	})

	return container
}
