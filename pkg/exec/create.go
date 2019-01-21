package exec

import (
	"context"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"regexp"
	"strconv"
	"time"
)

func createContainer(
	cli *client.Client, c *config.Configuration, sc *config.StartupConfiguration,
	env []string, mounts []mount.Mount) string {

	var command []string
	if len(sc.Args) > 0 {
		command = sc.Args
	} else {
		command = c.Command
	}

	var additionalGroups []string
	if sc.ShareDockerSocket && !sc.KeepUser {
		additionalGroups = append(additionalGroups, "docker")
	}

	if created, err := cli.ContainerCreate(
		context.TODO(), // TODO
		&container.Config{
			Image:        c.Image,
			Env:          env,
			User:         getUserAndGroup(),
			Cmd:          strslice.StrSlice(command),
			Tty:          c.Tty,
			OpenStdin:    c.StdinOpen,
			AttachStdin:  c.StdinOpen,
			AttachStdout: true,
			AttachStderr: true,
		},
		&container.HostConfig{
			AutoRemove: true,
			Privileged: sc.DesktopMode || c.Privileged, // TODO is this absolutely necessary for starting X ?
			Mounts:     mounts,
			GroupAdd:   additionalGroups,
		},
		&network.NetworkingConfig{},
		generateName(c),
	); err != nil {
		panic(err)
	} else {
		return created.ID
	}
}

func generateName(c *config.Configuration) string {
	name := c.Name

	if c.Name == "" {
		name = regexp.MustCompile("(?:.*/)?(.+?)(?::.*)?").ReplaceAllString(c.Image, "$1")
	}

	return name + "-" + strconv.Itoa(int(time.Now().Unix()))
}
