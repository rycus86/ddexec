package exec

import (
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/mattn/go-shellwords"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/convert"
	"github.com/rycus86/ddexec/pkg/debug"
)

func newContainerConfig(c *config.AppConfiguration, sc *config.StartupConfiguration, environment []string) *container.Config {
	var command []string
	if len(sc.Args) > 0 {
		command = sc.Args
	} else {
		command = getCommand(c)
	}

	var user string
	if !sc.KeepUser {
		user = getUserAndGroup()
	}

	var stopTimeout *int
	if c.StopTimeout != nil {
		stopTimeout = new(int)
		*stopTimeout = int(c.StopTimeout.Seconds())
	}

	var labels = map[string]string{
		"com.github.rycus86.ddexec.name":    c.Name,
		"com.github.rycus86.ddexec.version": config.GetVersion(),
	}
	for key, value := range c.Labels {
		labels[key] = value
	}

	exposed, _, err := nat.ParsePortSpecs(c.Ports)
	if err != nil {
		panic(err)
	}

	if debug.IsEnabled() && len(command) > 0 {
		fmt.Println("Running with command:", c.Command)
	}

	return &container.Config{
		Image:        c.Image,
		Env:          environment,
		User:         user,
		Cmd:          strslice.StrSlice(command),
		WorkingDir:   control.Target(c.WorkingDir, sc),
		Labels:       labels,
		Tty:          c.Tty,
		OpenStdin:    c.StdinOpen,
		AttachStdin:  c.StdinOpen,
		AttachStdout: true,
		AttachStderr: true,
		StopSignal:   c.StopSignal,
		StopTimeout:  stopTimeout,
		ExposedPorts: exposed,
	}
}

func getCommand(c *config.AppConfiguration) []string {
	cmd := convert.ToStringSlice(c.Command)
	if len(cmd) == 1 {
		if parsed, err := shellwords.Parse(cmd[0]); err != nil {
			panic(err)
		} else {
			return parsed
		}
	} else {
		return cmd
	}
}
