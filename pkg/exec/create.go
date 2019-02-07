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
	"github.com/mattn/go-shellwords"
	"github.com/pkg/errors"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/convert"
	"github.com/rycus86/ddexec/pkg/debug"
	"io/ioutil"
	"math/big"
	"path/filepath"
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
		generateName(c),
	); err != nil {
		panic(err)
	} else {
		return created.ID
	}
}

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
	}
}

func newHostConfig(c *config.AppConfiguration, sc *config.StartupConfiguration, mounts []mount.Mount) *container.HostConfig {
	additionalGroups := c.GroupAdd

	if !sc.KeepUser {
		hasDocker := false
		hasAudio := false
		hasVideo := false

		for _, gr := range additionalGroups {
			if gr == "docker" {
				hasDocker = true
			} else if gr == "audio" {
				hasAudio = true
			} else if gr == "video" {
				hasVideo = true
			}
		}

		if sc.ShareDockerSocket && !hasDocker {
			additionalGroups = append(additionalGroups, "docker")
		}
		if sc.ShareSound && !hasAudio {
			additionalGroups = append(additionalGroups, "audio")
		}
		if sc.ShareVideo && !hasVideo {
			additionalGroups = append(additionalGroups, "video")
		}
	}

	var devices []container.DeviceMapping
	var existingDevices = map[string]bool{}
	for _, device := range c.Devices {
		// TODO parse this properly
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        device,
			PathInContainer:   device,
			CgroupPermissions: "rwm",
		})

		existingDevices[device] = true
	}
	if sc.ShareSound && !existingDevices["/dev/snd"] {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        "/dev/snd",
			PathInContainer:   "/dev/snd",
			CgroupPermissions: "rwm",
		})
	}
	if sc.ShareVideo && !existingDevices["/dev/dri"] {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        "/dev/dri",
			PathInContainer:   "/dev/dri",
			CgroupPermissions: "rwm",
		})
	}
	if sc.ShareVideo && !existingDevices["/dev/video0"] {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        "/dev/video0",
			PathInContainer:   "/dev/video0",
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

	unitToBytes := func(value string) int64 {
		if value == "" {
			return 0
		} else if converted, err := units.RAMInBytes(value); err != nil {
			panic(err)
		} else {
			return converted
		}
	}

	var tmpfs = map[string]string{}
	var tmpfsConfig = convert.ToStringSlice(c.Tmpfs)
	if len(tmpfsConfig) > 0 {
		for _, item := range tmpfsConfig {
			if arr := strings.SplitN(item, ":", 2); len(arr) > 1 {
				tmpfs[arr[0]] = arr[1]
			} else {
				tmpfs[arr[0]] = ""
			}
		}
	}

	return &container.HostConfig{
		AutoRemove:  true,
		Privileged:  c.Privileged, // TODO is this absolutely necessary for starting X ?
		Mounts:      mounts,
		GroupAdd:    additionalGroups,
		SecurityOpt: securityOpts,
		CapAdd:      strslice.StrSlice(c.CapAdd),
		CapDrop:     strslice.StrSlice(c.CapDrop),
		NetworkMode: container.NetworkMode(c.NetworkMode), // TODO can be container:<x> or service:<x>
		IpcMode:     container.IpcMode(c.Ipc),
		PidMode:     container.PidMode(c.Pid),
		Tmpfs:       tmpfs,
		Resources: container.Resources{
			Devices:           devices,
			Memory:            unitToBytes(c.MemoryLimit),
			MemoryReservation: unitToBytes(c.MemoryReservation),
			MemorySwap:        unitToBytes(c.MemorySwap),
			MemorySwappiness:  c.MemorySwappiness,
			NanoCPUs:          parseCPUs(c.Cpus),
			CPUShares:         c.CpuShares,
			CPUPeriod:         c.CpuPeriod,
			CPUQuota:          c.CpuQuota,
			CpusetCpus:        c.CpusetCpus,
		},
	}
}

func generateName(c *config.AppConfiguration) string {
	name := c.Name

	if c.Name == "" {
		name = regexp.MustCompile("(?:.*/)?(.+?)(?::.*)?").ReplaceAllString(c.Image, "$1")
	}

	return name + "-" + strconv.Itoa(int(time.Now().Unix()))
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

// https://github.com/docker/cli/blob/9de1b162f/opts/opts.go#L372-L383
func parseCPUs(value string) int64 {
	if value == "" {
		return 0
	}
	cpu, ok := new(big.Rat).SetString(value)
	if !ok {
		panic(fmt.Errorf("cpus: failed to parse %v as a rational number", value))
	}
	nano := cpu.Mul(cpu, big.NewRat(1e9, 1))
	if !nano.IsInt() {
		panic(fmt.Errorf("cpus: value is too precise"))
	}
	return nano.Num().Int64()
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
