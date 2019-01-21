package exec

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/debug"
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
				if debug.IsEnabled() {
					fmt.Println("Received signal:", s)
				}

				cli.ContainerKill(context.TODO(), containerID, strconv.Itoa(int(s.(syscall.Signal))))
			}
		}
	}()
}
