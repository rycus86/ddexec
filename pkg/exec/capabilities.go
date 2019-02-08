package exec

import (
	"context"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"strings"
)

func loadDaemonCapabilities(cli *client.Client, sc *config.StartupConfiguration) {
	info, err := cli.Info(context.TODO()) // TODO
	if err != nil {
		panic(err)
	}

	for _, opt := range info.SecurityOptions {
		if strings.Contains(opt, "name=seccomp") {
			sc.DaemonHasSeccompSupport = true
		}
	}
}
