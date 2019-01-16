package exec

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	ddconfig "github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/files"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func Run(c *ddconfig.Configuration) int {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	cli.NegotiateAPIVersion(context.TODO()) // TODO

	i, _, err := cli.ImageInspectWithRaw(context.TODO(), c.Image)
	if err != nil {
		panic(err) // FIXME maybe just needs to pull the image
	}
	envPath := ""
	for _, item := range i.Config.Env {
		if strings.HasPrefix(item, "PATH=") {
			envPath = item
			break
		}
	}
	if strings.Contains(envPath, ":") {
		envPath = envPath + ":/usr/local/ddexec/bin"
	} else {
		envPath = "PATH=/usr/local/ddexec/bin"
	}

	env := []string{
		envPath,
	}
	if c.DesktopMode {
		env = append(env, "XAUTHORITY=/tmp/.server.xauth")
	} else {
		env = append(env, "DISPLAY="+os.Getenv("DISPLAY"))
		env = append(env, "XAUTHORITY="+getXauth())
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: "Xsocket",
			Target: "/tmp/.X11-unix",
		},
		{
			Type:   mount.TypeBind,
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		},
	}

	if c.DesktopMode {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/run/udev",
			Target: "/run/udev",
		})

		os.MkdirAll("/var/tmp/ddexec-log", 0777)
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: "/var/tmp/ddexec-log",
			Target: "/var/log",
		})

		os.MkdirAll(os.ExpandEnv("${HOME}/.ddexec"), 0777)
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: os.ExpandEnv("${HOME}/.ddexec"),
			Target: "/home/" + getUsername(), // TODO perhaps read this from /etc/passwd
		})

		os.MkdirAll(os.ExpandEnv("${HOME}/.ddexec.bin"), 0777)
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: os.ExpandEnv("${HOME}/.ddexec.bin"),
			Target: "/usr/local/ddexec/bin",
		})
	}

	created, err := cli.ContainerCreate(
		context.TODO(), // TODO
		&container.Config{
			Image: c.GetImage(),
			Env:   env,
			User:  getUserAndGroup(),
		},
		&container.HostConfig{
			AutoRemove: true,
			Privileged: c.DesktopMode || c.Privileged, // TODO is this absolutely necessary for starting X ?
			Mounts:     mounts,
			GroupAdd:   []string{"docker"},
		},
		&network.NetworkingConfig{},
		c.GetName()+"-"+strconv.Itoa(int(time.Now().Unix())),
	)
	if err != nil {
		panic(err)
	}

	// copy files
	passwdFiles := prepareUserAndGroupFiles(c)
	defer os.Remove(passwdFiles.Passwd)
	defer os.Remove(passwdFiles.Group)
	defer os.Remove(passwdFiles.Shadow)

	copyToContainer(cli, created.ID, passwdFiles.Passwd, "/etc/passwd")
	copyToContainer(cli, created.ID, passwdFiles.Group, "/etc/group")
	copyToContainer(cli, created.ID, passwdFiles.Shadow, "/etc/shadow")

	copyToContainer(cli, created.ID, getExecutable(), "/usr/local/bin/ddexec")

	if !c.DesktopMode {
		if err := prepareXauth(); err != nil {
			panic(err)
		}

		copyToContainer(cli, created.ID, getXauth(), getXauth())
	}

	// start the container
	err = cli.ContainerStart(
		context.TODO(), // TODO
		created.ID,
		types.ContainerStartOptions{},
	)
	if err != nil {
		panic(err)
	}

	// signal handlers
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel)

	go func() {
		for {
			select {
			case s := <-signalChannel:
				cli.ContainerKill(context.TODO(), created.ID, strconv.Itoa(int(s.(syscall.Signal))))
			}
		}
	}()

	// logs
	go func() {
		logs, err := cli.ContainerLogs(
			context.Background(),
			created.ID,
			types.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Follow:     true,
			})
		if err != nil {
			panic(err)
		}

		stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
	}()

	chWait, chErr := cli.ContainerWait(
		context.Background(),
		created.ID,
		container.WaitConditionNextExit)

	for {
		select {
		case w := <-chWait:
			if w.Error != nil {
				os.Stderr.WriteString(w.Error.Message)
			}

			rc, err := cli.ContainerLogs(
				context.TODO(), // TODO
				created.ID,
				types.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
				})
			if err != nil {
				fmt.Println(err)
			} else {
				io.Copy(os.Stdout, rc)
				rc.Close()
			}

			return int(w.StatusCode)
		case err := <-chErr:
			panic(err)
		}
	}
}

func copyToContainer(cli *client.Client, containerId, source, target string) {
	tar, err := createTar(source, filepath.Base(target))
	if err != nil {
		panic(err)
	}

	if err := cli.CopyToContainer(
		context.TODO(), // TODO
		containerId,
		filepath.Dir(target),
		tar,
		types.CopyToContainerOptions{}); err != nil {
		panic(err)
	}
}

func createTar(path, filename string) (io.Reader, error) {
	var b bytes.Buffer

	fi, err := os.Stat(path)
	if err != nil {
		panic(err)
	}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	tw := tar.NewWriter(&b)
	hdr := tar.Header{
		Name: filename,
		Mode: int64(fi.Mode().Perm()),
		Size: fi.Size(),
		Uid:  int(fi.Sys().(*syscall.Stat_t).Uid),
		Gid:  int(fi.Sys().(*syscall.Stat_t).Gid),
	}
	if err := tw.WriteHeader(&hdr); err != nil {
		panic(err)
	}

	if _, err = tw.Write(contents); err != nil {
		panic(err)
	}
	if err := tw.Close(); err != nil {
		panic(err)
	}

	return &b, nil
}

func prepareXauth() error {
	target := getXauth()

	if fi, err := os.Stat(target); err == nil && !fi.IsDir() && fi.Size() > 0 {
		return nil // already exists
	}

	return exec.Command("sh", "-c",
		"touch "+target+" && "+
			"xauth nlist "+os.Getenv("DISPLAY")+" | sed -e 's/^..../ffff/' | xauth -f "+target+" nmerge -",
	).Run()
}

func getXauth() string {
	s := os.Getenv("XAUTH")
	if s == "" {
		s = "/tmp/.docker.xauth"
	}
	return s
}

func prepareUserAndGroupFiles(c *ddconfig.Configuration) *files.PasswdFiles {
	if c.DesktopMode {
		passwd := files.CopyToTempfile("/etc/passwd")
		group := files.CopyToTempfile("/etc/group")
		shadow := files.WriteToTempfile(strings.TrimSpace(fmt.Sprintf(`
%s:!::0:99999:7:::
root:!::0:99999:7:::
`, getUsername())))

		files.ModifyFile(passwd, "(?m)^("+getUsername()+":.+:)[^:]*$", "$1/bin/sh")

		return &files.PasswdFiles{
			Passwd: passwd,
			Group:  group,
			Shadow: shadow,
		}
	} else {
		return &files.PasswdFiles{
			Passwd: "/etc/passwd",
			Group:  "/etc/group",
			Shadow: "/etc/shadow",
		}
	}
}

func getExecutable() string {
	e, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return e
}

func getUserAndGroup() string {
	return strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
}

func getUsername() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return u.Username
}
