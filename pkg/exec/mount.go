package exec

import (
	"github.com/docker/docker/api/types/mount"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"os"
	"strconv"
)

func prepareMounts(c *config.Configuration, sc *config.StartupConfiguration) []mount.Mount {
	var mountList []mount.Mount

	mountList = append(mountList, mount.Mount{
		Type:   mount.TypeBind,
		Source: control.GetDirectoryToShare(),
		Target: control.GetDirectoryToShare(),
	})

	if sc.ShareDockerSocket {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		})
	}

	if sc.UseHostX11 {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/tmp/.X11-unix",
			Target: "/tmp/.X11-unix",
		})
	} else if sc.DesktopMode || sc.ShareX11 {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeVolume,
			Source: "Xsocket",
			Target: "/tmp/.X11-unix",
		})
	}

	if sc.ShareDBus {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/run/dbus",
			Target: "/run/dbus",
		})
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/run/user/" + strconv.Itoa(os.Getuid()),
			Target: "/run/user/" + strconv.Itoa(os.Getuid()),
		})
	}

	if sc.ShareShm {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/dev/shm",
			Target: "/dev/shm",
		})
	}

	if sc.DesktopMode {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/run/udev",
			Target: "/run/udev",
		})

		if sc.XorgLogs != "" {
			mountList = append(mountList, mount.Mount{
				Type:   mount.TypeBind,
				Source: control.UnsafeEnsureSourceExists(sc.XorgLogs),
				Target: "/var/log",
			})
		}

	}

	if sc.ShareHomeDir {
		if sc.KeepUser {
			mountList = append(mountList, mount.Mount{
				Type:   mount.TypeBind,
				Source: control.EnsureSourceExists("$HOME"),
				Target: control.Target("$HOME", sc),
			})
		} else {
			mountList = append(mountList, mount.Mount{
				Type:   mount.TypeBind,
				Source: control.EnsureSourceExists("$HOME"),
				Target: control.Target("$HOME", sc),
			})
		}
	}

	if sc.ShareTools {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: control.EnsureSourceExists("${HOME}/../bin"),
			Target: "/usr/local/ddexec/bin",
		})
	}

	for _, vc := range c.Volumes {
		if vc.Type == string(mount.TypeBind) {
			control.EnsureSourceExists(vc.Source)
		}

		mountList = append(mountList, mount.Mount{
			Type:     mount.Type(vc.Type),
			Source:   control.Source(vc.Source),
			Target:   control.Target(vc.Target, sc),
			ReadOnly: vc.ReadOnly,
		})
	}

	return mountList
}
