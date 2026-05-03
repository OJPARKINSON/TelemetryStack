package containers

import (
	"context"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
)

func removeContainerByName(ctx context.Context, name string) {
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		return
	}
	defer provider.Close()

	cli := provider.Client()
	if cli == nil {
		return
	}
	_ = cli.ContainerRemove(ctx, name, container.RemoveOptions{Force: true, RemoveVolumes: true})
}
