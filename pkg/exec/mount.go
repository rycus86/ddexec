package exec

import (
	"github.com/docker/docker/api/types/mount"
	"github.com/rycus86/ddexec/pkg/config"
	"os"
	"strconv"
)

// FIXME mkdir for mounts won't work from inside a container
func prepareMounts(c *config.Configuration, sc *config.StartupConfiguration) []mount.Mount {
	var mounts []mount.Mount

	if sc.ShareDockerSocket {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		})
	}

	if sc.UseHostX11 {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/tmp/.X11-unix",
			Target: "/tmp/.X11-unix",
		})
	} else if sc.DesktopMode || sc.ShareX11 {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: "Xsocket",
			Target: "/tmp/.X11-unix",
		})
	}

	if sc.ShareDBus {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/run/dbus",
			Target: "/run/dbus",
		})
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/run/user/" + strconv.Itoa(os.Getuid()),
			Target: "/run/user/" + strconv.Itoa(os.Getuid()),
		})
	}

	if sc.DesktopMode {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/run/udev",
			Target: "/run/udev",
		})

		if sc.XorgLogs != "" {
			os.MkdirAll(sc.XorgLogs, 0777)
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: sc.XorgLogs,
				Target: "/var/log",
			})
		}

		if sc.SharedHomeDir {
			os.MkdirAll(os.ExpandEnv("${HOME}/.ddexec"), 0777)

			if sc.KeepUser {
				home := sc.ImageHome
				if home == "" {
					home = "/home/" + sc.ImageUser
				}

				mounts = append(mounts, mount.Mount{
					Type:   mount.TypeBind,
					Source: os.ExpandEnv("${HOME}/.ddexec"),
					Target: home, // TODO perhaps read this from /etc/passwd
				})

			} else {
				username := os.Getenv("USER")
				if username == "" {
					username = getUsername()
				}

				mounts = append(mounts, mount.Mount{
					Type:   mount.TypeBind,
					Source: os.ExpandEnv("${HOME}/.ddexec"),
					Target: "/home/" + username, // TODO perhaps read this from /etc/passwd
				})
			}
		}

		if sc.SharedTools {
			os.MkdirAll(os.ExpandEnv("${HOME}/.ddexec.bin"), 0777)
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: os.ExpandEnv("${HOME}/.ddexec.bin"),
				Target: "/usr/local/ddexec/bin",
			})
		}
	}

	for _, vc := range c.Volumes {
		if vc.Type == string(mount.TypeBind) {
			if _, err := os.Stat(vc.Source); err != nil && os.IsNotExist(err) {
				os.MkdirAll(os.ExpandEnv(vc.Source), 0777)
			}
		}

		mounts = append(mounts, mount.Mount{
			Type:     mount.Type(vc.Type),
			Source:   os.ExpandEnv(vc.Source),
			Target:   vc.Target,
			ReadOnly: vc.ReadOnly,
		})
	}

	return mounts
}
