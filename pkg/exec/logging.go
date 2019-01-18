package exec

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"os"
)

func setupStreams(cli *client.Client, containerID string) {
	go func() {
		logs, err := cli.ContainerLogs(
			context.Background(),
			containerID,
			types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Follow:     true,
			})
		if err != nil {
			panic(err)
		}

		stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
	}()
}
