package exec

import (
	"context"
	"github.com/docker/docker/client"
)

func newClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	cli.NegotiateAPIVersion(context.TODO()) // TODO

	return cli
}
