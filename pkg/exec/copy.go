package exec

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

func copyFiles(cli *client.Client, containerID string, sc *config.StartupConfiguration) {
	if !sc.KeepUser {
		passwdFiles := prepareUserAndGroupFiles(sc)
		if passwdFiles.Temporary {
			defer os.Remove(passwdFiles.Passwd)
			defer os.Remove(passwdFiles.Group)
		}
		// always delete the made-up /etc/shadow file
		defer os.Remove(passwdFiles.Shadow)

		copyToContainer(cli, containerID, passwdFiles.Passwd, "/etc/passwd")
		copyToContainer(cli, containerID, passwdFiles.Group, "/etc/group")
		copyToContainer(cli, containerID, passwdFiles.Shadow, "/etc/shadow")
	}

	copyToContainer(cli, containerID, getExecutable(), "/usr/local/bin/ddexec")

	if !sc.DesktopMode {
		if err := prepareXauth(); err != nil {
			panic(err)
		}

		copyToContainer(cli, containerID, getXauth(), getXauth())
	}
}

func copyToContainer(cli *client.Client, containerId, source, target string) {
	if debug.IsEnabled() {
		fmt.Println("Copying", source, "to", target, "...")
	}

	tarFile, err := createTar(source, filepath.Base(target))
	if err != nil {
		panic(err)
	}

	if err := cli.CopyToContainer(
		context.TODO(), // TODO
		containerId,
		filepath.Dir(target),
		tarFile,
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

func getExecutable() string {
	e, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return e
}
