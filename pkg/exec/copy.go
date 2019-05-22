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
	"syscall"
)

type fileToCopy struct {
	Source string
	Target string

	Contents []byte
	Header   *tar.Header
}

func copyFiles(cli *client.Client, containerID string, sc *config.StartupConfiguration) {
	var toCopy []fileToCopy

	if !sc.KeepUser {
		passwdFiles := prepareUserAndGroupFiles(sc)
		if passwdFiles.Temporary {
			defer os.Remove(passwdFiles.Passwd)
			defer os.Remove(passwdFiles.Group)
		}
		// always delete the made-up /etc/shadow file
		defer os.Remove(passwdFiles.Shadow)

		toCopy = append(toCopy, fileToCopy{Source: passwdFiles.Passwd, Target: "/etc/passwd"})
		toCopy = append(toCopy, fileToCopy{Source: passwdFiles.Group, Target: "/etc/group"})
		toCopy = append(toCopy, fileToCopy{Source: passwdFiles.Shadow, Target: "/etc/shadow"})
	}

	if sc.ShareTools {
		toCopy = append(toCopy, fileToCopy{Source: getExecutable(), Target: "/usr/local/bin/ddexec"})
	}

	// TODO condition?
	toCopy = append(toCopy, fileToCopy{Source: getExecutable(), Target: "/usr/local/ddexec-xdg/bin/xdg-open"})

	if !sc.DesktopMode {
		if err := prepareXauth(); err != nil {
			panic(err)
		}

		toCopy = append(toCopy, fileToCopy{Source: getXauth(), Target: getXauth()})
	}

	if err := copyToContainer(cli, containerID, "/", toCopy...); err != nil {
		panic(err)
	}
}

func copyToContainer(cli *client.Client, containerId string, dstPath string, files ...fileToCopy) error {
	if debug.IsEnabled() {
		for _, file := range files {
			fmt.Println("Copying", file.Source, "to", file.Target, "...")
		}
	}

	tarFile, err := createTar(files...)
	if err != nil {
		return err
	}

	if err := cli.CopyToContainer(
		context.TODO(), // TODO
		containerId,
		dstPath,
		tarFile,
		types.CopyToContainerOptions{}); err != nil {
		return err
	}

	return nil
}

func createTar(files ...fileToCopy) (io.Reader, error) {
	var b bytes.Buffer

	tw := tar.NewWriter(&b)

	for _, file := range files {
		var contents []byte
		var hdr *tar.Header

		if file.Contents != nil {
			contents = file.Contents
			hdr = file.Header

		} else {
			fi, err := os.Stat(file.Source)
			if err != nil {
				return nil, err
			}

			contents, err = ioutil.ReadFile(file.Source)
			if err != nil {
				return nil, err
			}

			hdr = &tar.Header{
				Name: file.Target,
				Mode: int64(fi.Mode().Perm()),
				Size: fi.Size(),
				Uid:  int(fi.Sys().(*syscall.Stat_t).Uid),
				Gid:  int(fi.Sys().(*syscall.Stat_t).Gid),
			}
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return nil, err
		}

		if _, err := tw.Write(contents); err != nil {
			return nil, err
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
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
