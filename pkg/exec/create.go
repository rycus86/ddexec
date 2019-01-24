package exec

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/rycus86/ddexec/pkg/config"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func createContainer(
	cli *client.Client, c *config.Configuration, sc *config.StartupConfiguration,
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

func newContainerConfig(c *config.Configuration, sc *config.StartupConfiguration, environment []string) *container.Config {
	var command []string
	if len(sc.Args) > 0 {
		command = sc.Args
	} else {
		command = c.Command
	}

	var user string
	if !sc.KeepUser {
		user = getUserAndGroup()
	}

	return &container.Config{
		Image:        c.Image,
		Env:          environment,
		User:         user,
		Cmd:          strslice.StrSlice(command),
		Tty:          c.Tty,
		OpenStdin:    c.StdinOpen,
		AttachStdin:  c.StdinOpen,
		AttachStdout: true,
		AttachStderr: true,
	}
}

func newHostConfig(c *config.Configuration, sc *config.StartupConfiguration, mounts []mount.Mount) *container.HostConfig {
	var additionalGroups []string
	if sc.ShareDockerSocket && !sc.KeepUser {
		additionalGroups = append(additionalGroups, "docker")
	}

	var devices []container.DeviceMapping
	for _, device := range c.Devices {
		// TODO parse this properly
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        device,
			PathInContainer:   device,
			CgroupPermissions: "rwm",
		})
	}

	var securityOpts []string
	if len(c.SecurityOpts) > 0 {
		if so, err := parseSecurityOpts(c.SecurityOpts); err != nil {
			panic(err)
		} else {
			securityOpts = so
		}
	}

	var memLimit int64
	if c.MemLimit != "" {
		if l, err := units.RAMInBytes(c.MemLimit); err != nil {
			panic(err)
		} else {
			memLimit = l
		}
	}

	return &container.HostConfig{
		AutoRemove:  true,
		Privileged:  sc.DesktopMode || c.Privileged, // TODO is this absolutely necessary for starting X ?
		Mounts:      mounts,
		GroupAdd:    additionalGroups,
		SecurityOpt: securityOpts,
		CapAdd:      strslice.StrSlice(c.CapAdd),
		CapDrop:     strslice.StrSlice(c.CapDrop),
		IpcMode:     container.IpcMode(c.Ipc),
		Resources: container.Resources{
			Devices: devices,
			Memory:  memLimit,
		},
	}
}

func generateName(c *config.Configuration) string {
	name := c.Name

	if c.Name == "" {
		name = regexp.MustCompile("(?:.*/)?(.+?)(?::.*)?").ReplaceAllString(c.Image, "$1")
	}

	return name + "-" + strconv.Itoa(int(time.Now().Unix()))
}

// https://github.com/docker/cli/blob/9de1b162f/cli/command/container/opts.go#L673-L697
func parseSecurityOpts(securityOpts []string) ([]string, error) {
	for key, opt := range securityOpts {
		con := strings.SplitN(opt, "=", 2)
		if len(con) == 1 && con[0] != "no-new-privileges" {
			if strings.Contains(opt, ":") {
				con = strings.SplitN(opt, ":", 2)
			} else {
				return securityOpts, errors.Errorf("Invalid --security-opt: %q", opt)
			}
		}
		if con[0] == "seccomp" && con[1] != "unconfined" {
			f, err := ioutil.ReadFile(filepath.Clean(con[1]))
			if err != nil {
				return securityOpts, errors.Errorf("opening seccomp profile (%s) failed: %v", con[1], err)
			}
			b := bytes.NewBuffer(nil)
			if err := json.Compact(b, f); err != nil {
				return securityOpts, errors.Errorf("compacting json for seccomp profile (%s) failed: %v", con[1], err)
			}
			securityOpts[key] = fmt.Sprintf("seccomp=%s", b.Bytes())
		}
	}

	return securityOpts, nil
}
