package exec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/convert"
	"github.com/rycus86/ddexec/pkg/debug"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"
)

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
		if deviceExists(device) {
			// TODO parse this properly
			devices = append(devices, container.DeviceMapping{
				PathOnHost:        device,
				PathInContainer:   device,
				CgroupPermissions: "rwm",
			})

			existingDevices[device] = true
		}
	}
	if sc.ShareSound && deviceExists("/dev/snd") && !existingDevices["/dev/snd"] {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        "/dev/snd",
			PathInContainer:   "/dev/snd",
			CgroupPermissions: "rwm",
		})
	}
	if sc.ShareVideo && deviceExists("/dev/dri") && !existingDevices["/dev/dri"] {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        "/dev/dri",
			PathInContainer:   "/dev/dri",
			CgroupPermissions: "rwm",
		})
	}
	if sc.ShareVideo && deviceExists("/dev/video0") && !existingDevices["/dev/video0"] {
		devices = append(devices, container.DeviceMapping{
			PathOnHost:        "/dev/video0",
			PathInContainer:   "/dev/video0",
			CgroupPermissions: "rwm",
		})
	}

	var securityOpts []string
	if len(c.SecurityOpts) > 0 {
		if !sc.DaemonHasSeccompSupport {
			fmt.Println("WARNING: No seccomp support detected")
		} else if so, err := parseSecurityOpts(c.SecurityOpts); err != nil {
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

	_, ports, err := nat.ParsePortSpecs(c.Ports)
	if err != nil {
		panic(err)
	}

	return &container.HostConfig{
		AutoRemove:     true,
		Privileged:     c.Privileged, // TODO is this absolutely necessary for starting X ?
		ReadonlyRootfs: c.ReadOnly,
		Mounts:         mounts,
		GroupAdd:       additionalGroups,
		SecurityOpt:    securityOpts,
		CapAdd:         strslice.StrSlice(c.CapAdd),
		CapDrop:        strslice.StrSlice(c.CapDrop),
		NetworkMode:    container.NetworkMode(c.NetworkMode), // TODO can be container:<x> or service:<x>
		IpcMode:        container.IpcMode(c.Ipc),
		PidMode:        container.PidMode(c.Pid),
		Tmpfs:          tmpfs,
		OomScoreAdj:    c.OomScoreAdj,
		ShmSize:        unitToBytes(c.ShmSize),
		Init:           c.Init,
		PortBindings:   ports,
		Resources: container.Resources{
			OomKillDisable:    c.OomKillDisable,
			PidsLimit:         c.PidsLimit,
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

func deviceExists(path string) bool {
	if exists, err := control.CheckDevice(path); err != nil {
		if debug.IsEnabled() {
			fmt.Println("Failed to check if device at", path, "exists")
		}
		return false
	} else {
		return exists
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
