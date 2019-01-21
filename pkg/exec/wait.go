package exec

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"os"
)

func waitForExit(cli *client.Client, containerID string) int {
	chWait, chErr := cli.ContainerWait(
		context.Background(),
		containerID,
		container.WaitConditionNotRunning)

	for {
		select {
		case w := <-chWait:
			if w.Error != nil {
				os.Stderr.WriteString(w.Error.Message)
			}

			return int(w.StatusCode)

		case err := <-chErr:
			panic(err)
		}
	}
}
