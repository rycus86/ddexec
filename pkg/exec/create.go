package exec

import (
	"context"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"regexp"
	"strconv"
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
		generateName(c),
	); err != nil {
		panic(err)
	} else {
		return created.ID
	}
}

func generateName(c *config.AppConfiguration) string {
	name := c.Name

	if c.Name == "" {
		name = regexp.MustCompile("(?:.*/)?(.+?)(?::.*)?").ReplaceAllString(c.Image, "$1")
	}

	return name + "-" + strconv.Itoa(int(time.Now().Unix()))
}
