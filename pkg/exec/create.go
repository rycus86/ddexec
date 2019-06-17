package exec

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/env"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func createContainer(
	cli *client.Client, c *config.AppConfiguration, sc *config.StartupConfiguration,
	env []string, mounts []mount.Mount) string {

	if created, err := cli.ContainerCreate(
		context.TODO(), // TODO
		newContainerConfig(c, sc, env),
		newHostConfig(c, sc, mounts),
		&network.NetworkingConfig{},
		generateName(cli, c),
	); err != nil {
		panic(err)
	} else {
		return created.ID
	}
}

func generateName(cli *client.Client, c *config.AppConfiguration) string {
	name := c.Name

	if c.Name == "" {
		name = regexp.MustCompile("(?:.*/)?(.+?)(?::.*)?").ReplaceAllString(c.Image, "$1")
	}

	if env.IsSet("DDEXEC_UNIQUE_NAMES") {
		return name + "-" + strconv.Itoa(int(time.Now().Unix()))
	} else {
		containers, err := cli.ContainerList(context.TODO(), types.ContainerListOptions{
			Filters: filters.NewArgs(filters.Arg("name", name)),
			All:     true,
		})
		if err != nil {
			panic(err)
		}

		baseName := name
		i := 1

		for ; hasContainer(containers, name); i++ {
			name = baseName + "-" + strconv.Itoa(i)
		}

		return name
	}
}

func hasContainer(containers []types.Container, name string) bool {
	for _, container := range containers {
		for _, cn := range container.Names {
			if strings.TrimPrefix(cn, "/") == name {
				return true
			}
		}
	}

	return false
}
