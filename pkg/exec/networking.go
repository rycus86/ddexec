package exec

import (
	"archive/tar"
	"context"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"io/ioutil"
	"strings"
)

func setupNetworking(cli *client.Client, containerID string, sc *config.StartupConfiguration) {
	gwAddress := getGatewayAddress(cli, containerID)

	etcHostsFile := getFromContainer(cli, containerID, "/etc/hosts")

	contents := string(etcHostsFile.Contents) + "\n"

	contents += "\n" + gwAddress + "  ddexec.local" + "\n"

	for _, mapping := range sc.Hostnames {
		parts := strings.Split(mapping, "=")
		hostname, target := parts[0], parts[1]

		if target == "host" {
			target = gwAddress
		}

		contents += "\n" + target + "  " + hostname
	}

	etcHostsFile.Contents = []byte(contents)
	etcHostsFile.Header.Size = int64(len(etcHostsFile.Contents))

	etcHostsFile.Source = "/etc/hosts"
	etcHostsFile.Target = "/etc/hosts"

	copyToContainer(cli, containerID, "/etc", etcHostsFile)
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
		Header: hdr,
	}
}
