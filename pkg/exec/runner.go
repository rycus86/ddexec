package exec

import (
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
)

func Run(c *config.Configuration, sc *config.StartupConfiguration) int {
	control.StartServerIfNecessary()

	cli := newClient()
	defer cli.Close()

	prepareAndProcessImage(cli, c, sc)

	env := prepareEnvironment(sc)
	mounts := prepareMounts(c, sc)

	containerID := createContainer(cli, c, sc, env, mounts)
	copyFiles(cli, containerID, sc)

	if closeStreams := setupStreams(cli, containerID, c); closeStreams != nil {
		defer closeStreams()
	}

	startContainer(cli, containerID)

	setupSignalHandlers(cli, containerID)

	return waitForExit(cli, containerID)
}
