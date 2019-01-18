package exec

import (
	"context"
	"github.com/docker/docker/client"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func setupSignalHandlers(cli *client.Client, containerID string) {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel)

	go func() {
		for {
			select {
			case s := <-signalChannel:
				cli.ContainerKill(context.TODO(), containerID, strconv.Itoa(int(s.(syscall.Signal))))
			}
		}
	}()
}
