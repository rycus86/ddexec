package exec

import (
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-units"
	"github.com/mitchellh/mapstructure"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/volume"
	"os"
	"strconv"
	"strings"
)

func prepareMounts(c *config.AppConfiguration, sc *config.StartupConfiguration) []mount.Mount {
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
	} else if sc.ShareX11 {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeVolume,
			Source: "Xsocket",
			Target: "/tmp/.X11-unix",
		})
	}

	if sc.ShareDBus {
		if sc.UseHostDBus {
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
		} else {
			mountList = append(mountList, mount.Mount{
				Type:   mount.TypeVolume,
				Source: "Xdbus",
				Target: "/run/dbus",
			})
			mountList = append(mountList, mount.Mount{
				Type:   mount.TypeVolume,
				Source: "XdbusUser",
				Target: "/run/user/" + strconv.Itoa(os.Getuid()),
			})
		}
	}

	if sc.ShareShm {
		mountList = append(mountList, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/dev/shm",
			Target: "/dev/shm",
		})
	}

	if sc.DesktopMode {
		if udev, err := os.Stat("/run/udev"); err == nil && udev.IsDir() {
			mountList = append(mountList, mount.Mount{
				Type:   mount.TypeBind,
				Source: "/run/udev",
				Target: "/run/udev",
			})
		}

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
			Source: control.UnsafeEnsureSourceExists(control.Source("${HOME}/../bin")),
			Target: "/usr/local/ddexec/bin",
		})
	}

	for _, v := range parseVolumes(c.Volumes) {
		if v.GetMountType() == mount.TypeBind {
			src := v.Source

			v.Source = control.Source(src)
			control.EnsureSourceExists(src)
		}

		mnt := mount.Mount{
			Type:     v.GetMountType(),
			Source:   v.Source,
			Target:   control.Target(v.Target, sc),
			ReadOnly: v.IsReadOnly(),
		}

		if v.Bind.Propagation != "" {
			mnt.BindOptions = &mount.BindOptions{
				Propagation: mount.Propagation(v.Bind.Propagation),
			}
		}

		if v.Volume.NoCopy {
			mnt.VolumeOptions = &mount.VolumeOptions{
				NoCopy: true,
			}
		}

		if v.Tmpfs.Size != "" {
			size, err := units.FromHumanSize(v.Tmpfs.Size)
			if err != nil {
				panic(err)
			}

			mnt.TmpfsOptions = &mount.TmpfsOptions{
				SizeBytes: size,
			}
		}

		mountList = append(mountList, mnt)
	}

	if debug.IsEnabled() {
		for _, m := range mountList {
			fmt.Printf("mount: %+v\n", m)
		}
	}

	return mountList
}

func parseVolumes(vArr []interface{}) []*volume.Volume {
	if len(vArr) == 0 {
		return []*volume.Volume{}
	}

	converted := make([]*volume.Volume, len(vArr), len(vArr))

	for idx, item := range vArr {
		if asString, ok := item.(string); ok {
			v := volume.Volume{}
			parts := strings.Split(asString, ":")

			switch len(parts) {
			case 1:
				v.Target = parts[0]
			case 2:
				v.Source = parts[0]
				v.Target = parts[1]
			case 3:
				v.Source = parts[0]
				v.Target = parts[1]
				v.Mode = parts[2]
			}

			if strings.HasPrefix(control.Source(v.Source), "/") {
				v.Type = string(mount.TypeBind)
			} else {
				v.Type = string(mount.TypeVolume)
			}

			converted[idx] = &v
		} else {
			var v volume.Volume

			err := mapstructure.Decode(item, &v)
			if err != nil {
				panic(err)
			}

			converted[idx] = &v
		}
	}

	return converted
}
