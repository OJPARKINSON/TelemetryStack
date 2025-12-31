package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartRabbitMQ(t *testing.T, ctx context.Context) *rabbitmq.RabbitMQContainer {

	container, err := rabbitmq.Run(ctx, "rabbitmq:4.2-management",
		testcontainers.WithExposedPorts("5672:5672", "15672:15672", "15692:15692"),
		testcontainers.WithMounts(
			testcontainers.BindMount("/Users/op/Documents/IRacing-Display/config/definitions.json",
				testcontainers.ContainerMountTarget("/etc/rabbitmq/definitions.json")),
		),
		testcontainers.WithEnv(map[string]string{
			"RABBITMQ_DEFAULT_USER":                "guest",
			"RABBITMQ_DEFAULT_PASS":                "changeme",
			"RABBITMQ_MANAGEMENT_LOAD_DEFINITIONS": "/etc/rabbitmq/definitions.json",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5672"),
			// wait.ForLog("Ready to accept connections"),
		),
	)

	if err != nil {
		fmt.Println("Error running RabbitMQ server, ", err)
	}

	return container
}
