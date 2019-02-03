package exec

import (
	"context"
	"github.com/rycus86/ddexec/pkg/config"
)

func Run(c *config.AppConfiguration, sc *config.StartupConfiguration) (chan int, func()) {
	cli := newClient()
	defer func() {
		if err := recover(); err != nil {
			cli.Close()
			panic(err)
		}
	}()

	prepareAndProcessImage(cli, c, sc)

	env := prepareEnvironment(c, sc)
	mounts := prepareMounts(c, sc)

	containerID := createContainer(cli, c, sc, env, mounts)
	copyFiles(cli, containerID, sc)

	closeStreams := setupStreams(cli, containerID, c)

	startContainer(cli, containerID)

	setupSignalHandlers(cli, containerID)
	monitorTtySize(cli, containerID, c)

	waitChan := make(chan int, 1)

	go func() {
		defer cli.Close()
		if closeStreams != nil {
			defer closeStreams()
		}

		exitCode := waitForExit(cli, containerID)
		waitChan <- exitCode
	}()

	return waitChan, func() {
		cli.ContainerStop(context.TODO(), containerID, nil)
	}
}
