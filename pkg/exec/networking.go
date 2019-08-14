package exec

import (
	"context"
	"errors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rycus86/ddexec/pkg/config"
	"strings"
)

func prepareExtraHosts(cli *client.Client, sc *config.StartupConfiguration) []string {
	var extras []string

	gwAddress := findBridgeGatewayAddress(cli)

	extras = append(extras, "ddexec.local:"+gwAddress)

	for _, mapping := range sc.Hostnames {
		if !strings.Contains(mapping, ":") {
			panic(errors.New("illegal host mapping: " + mapping))
		}

		parts := strings.Split(mapping, ":")
		hostname, target := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

		if target == "host" {
			target = gwAddress
		}

		extras = append(extras, hostname+":"+target)
	}

	return extras
}

func findBridgeGatewayAddress(cli *client.Client) string {
	network, err := cli.NetworkInspect(context.Background(), "bridge", types.NetworkInspectOptions{
		Scope: "local",
	})
	if err != nil {
		panic(err)
	}

	configs := network.IPAM.Config
	if len(configs) == 0 {
		panic(errors.New("no IPAM config found on the " + network.ID + " network"))
	}

	return configs[0].Gateway
}
