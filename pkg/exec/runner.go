package exec

import (
	"context"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/env"
	"github.com/rycus86/ddexec/pkg/xdgopen"
)

func Run(c *config.AppConfiguration, sc *config.StartupConfiguration) (chan int, func()) {
	cli := newClient()
	defer func() {
		if err := recover(); err != nil {
			cli.Close()
			panic(err)
		}
	}()

	debug.LogTime("clientReady")

	loadDaemonCapabilities(cli, sc)

	debug.LogTime("daemonCapabilities")

	prepareAndProcessImage(cli, c, sc)

	debug.LogTime("prepareImage")

	if env.IsSet("DDEXEC_IMAGE_ONLY") {
		return nil, nil
	}

	checkStreams(sc)

	debug.LogTime("checkStreams")

	environment := prepareEnvironment(c, sc)

	debug.LogTime("prepareEnvironment")

	mounts := prepareMounts(c, sc)

	debug.LogTime("prepareMounts")

	extraHosts := prepareExtraHosts(cli, sc)

	debug.LogTime("prepareExtraHosts")

	containerID := createContainer(cli, c, sc, environment, mounts, extraHosts)

	debug.LogTime("createContainer")

	copyFiles(cli, containerID, sc)

	debug.LogTime("copyFiles")

	closeStreams := setupStreams(cli, containerID, c, sc)

	debug.LogTime("setupStreams")

	startContainer(cli, containerID)

	debug.LogTime("startContainer")

	setupSignalHandlers(cli, containerID)

	debug.LogTime("setupSignals")

	xdgopen.Register(containerID, sc)

	debug.LogTime("xdgopen.Register")

	monitorTtySize(cli, containerID, c, sc)

	debug.LogTime("monitorTty")

	waitChan := make(chan int, 1)

	go func() {
		defer cli.Close()
		if closeStreams != nil {
			defer closeStreams()
		}

		exitCode := waitForExit(cli, containerID)

		debug.LogTime("waitForExit")

		xdgopen.Clear(containerID)

		debug.LogTime("xdgopen.Clear")

		waitChan <- exitCode
	}()

	return waitChan, func() {
		debug.LogTime("closerStart")

		cli.ContainerStop(context.Background(), containerID, nil)

		debug.LogTime("containerStop")

		// TODO maybe this is unnecessary
		if selfId := getSelfContainerId(); selfId != "" {
			restoreTtySize(cli, selfId)

			debug.LogTime("restoryTtySize")
		}
	}
}
