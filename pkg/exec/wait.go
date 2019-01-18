package exec

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"io"
	"os"
)

func waitForExit(cli *client.Client, containerID string) int {
	chWait, chErr := cli.ContainerWait(
		context.Background(),
		containerID,
		container.WaitConditionNextExit)

	for {
		select {
		case w := <-chWait:
			if w.Error != nil {
				os.Stderr.WriteString(w.Error.Message)
			}

			rc, err := cli.ContainerLogs(
				context.TODO(), // TODO
				containerID,
				types.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
				})
			if err != nil {
				fmt.Println(err)
			} else {
				io.Copy(os.Stdout, rc)
				rc.Close()
			}

			return int(w.StatusCode)

		case err := <-chErr:
			panic(err)
		}
	}
}
