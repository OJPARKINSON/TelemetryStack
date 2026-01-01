package containers

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

func CreateNetwork(ctx context.Context) (*testcontainers.DockerNetwork, error) {
	network, err := network.New(ctx)
	return network, err
}
