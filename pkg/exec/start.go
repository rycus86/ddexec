package exec

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func startContainer(cli *client.Client, containerID string) {
	if err := cli.ContainerStart(
		context.Background(),
		containerID,
		types.ContainerStartOptions{},
	); err != nil {
		panic(err)
	}
}
