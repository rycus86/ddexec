package exec

import (
	"github.com/rycus86/ddexec/pkg/config"
)

func Run(c *config.Configuration, sc *config.StartupConfiguration) int {
	cli := newClient()
	defer cli.Close()

	prepareAndProcessImage(cli, c, sc)

	env := prepareEnvironment(sc)
	mounts := prepareMounts(c, sc)

	containerID := createContainer(cli, c, sc, env, mounts)
	copyFiles(cli, containerID, sc)

	startContainer(cli, containerID)

	setupSignalHandlers(cli, containerID)
	setupStreams(cli, containerID)

	return waitForExit(cli, containerID)
}
