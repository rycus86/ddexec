package exec

import (
	"archive/tar"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"io/ioutil"
	"strings"
)

func setupNetworking(cli *client.Client, containerID string, sc *config.StartupConfiguration) {
	gwAddress := getGatewayAddress(cli, containerID)

	etcHostsFile := getFromContainer(cli, containerID, "/etc/hosts")
	comment := "\n\n# Changed by ddexec with copy"
	toAppend := "\n\n" + gwAddress + "  ddexec.local" + "\n\n"

	for _, mapping := range sc.Hostnames {
		parts := strings.Split(mapping, "=")
		hostname, target := parts[0], parts[1]

		if target == "host" {
			target = gwAddress
		}

		toAppend += target + "  " + hostname + "\n"
	}

	etcHostsFile.Contents = []byte(string(etcHostsFile.Contents) + comment + toAppend)
	etcHostsFile.Header.Size = int64(len(etcHostsFile.Contents))

	etcHostsFile.Source = "modified:/etc/hosts"
	etcHostsFile.Target = "/etc/hosts"

	if err := copyToContainer(cli, containerID, "/etc", etcHostsFile); err != nil {
		if debug.IsEnabled() {
			fmt.Println("Failed to copy /etc/hosts into the container:", err)
		}

		if execErr := tryWriteEtcHosts(cli, containerID, toAppend); execErr != nil {
			if debug.IsEnabled() {
				fmt.Println("Failed to write /etc/hosts with sh+echo:", execErr)
			}

			panic(err)
		}
	}
}

func tryWriteEtcHosts(cli *client.Client, containerID string, toAppend string) error {
	comment := "\n\n# Changed by ddexec with sh+echo"

	response, err := cli.ContainerExecCreate(context.TODO(), containerID, types.ExecConfig{
		User: "0",
		Cmd:  []string{"sh", "-c", "echo '" + comment + toAppend + "' >> /etc/hosts"},
	})
	if err != nil {
		return err
	}

	return cli.ContainerExecStart(context.TODO(), response.ID, types.ExecStartCheck{Detach: true})
}

func getGatewayAddress(cli *client.Client, containerID string) string {
	i, err := cli.ContainerInspect(context.TODO(), containerID)
	if err != nil {
		panic(err)
	}

	return i.NetworkSettings.Gateway
}

func getFromContainer(cli *client.Client, containerID string, path string) fileToCopy {
	reader, _, err := cli.CopyFromContainer(context.TODO(), containerID, path)
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	tr := tar.NewReader(reader)
	hdr, err := tr.Next()

	contents, err := ioutil.ReadAll(tr)
	if err != nil {
		panic(err)
	}

	return fileToCopy{
		Contents: contents,
		Header:   hdr,
	}
}
