package xdgexec

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"os"
	"strings"
)

func ExecInContainer(containerId string, command string) (bool, error) {
	fmt.Println("exec in", containerId, ">", command)

	// TODO dupe of exec/client.go

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false, err
	}
	defer cli.Close()

	cli.NegotiateAPIVersion(context.TODO()) // TODO

	// TODO /bin/sh -c needs special characters escaped
	command = strings.ReplaceAll(command, "&", "\\&")

	exec, err := cli.ContainerExecCreate(context.TODO(), containerId, types.ExecConfig{
		// TODO using /bin/sh for now
		Cmd:          []string{"/bin/sh", "-c", command},
		Detach:       false,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		fmt.Println("exec create failed:", err)
		return false, err
	}

	fmt.Println("exec created OK")

	resp, err := cli.ContainerExecAttach(context.TODO(), exec.ID, types.ExecStartCheck{})
	if err != nil {
		fmt.Println("exec attach failed:", err)
		return false, err
	}
	defer resp.Close()

	fmt.Println("exec attached OK")

	if err = cli.ContainerExecStart(context.TODO(), exec.ID, types.ExecStartCheck{}); err != nil {
		fmt.Println("exec start failed:", err)
		return false, err
	}

	fmt.Println("exec started OK")

	stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader)

	fmt.Println("exec read OK")

	inspect, err := cli.ContainerExecInspect(context.TODO(), exec.ID)
	if err != nil {
		fmt.Println("exec inspect failed:", err)
		return false, err
	}

	fmt.Println("exec inspect OK, exit:", inspect.ExitCode)

	return inspect.ExitCode == 0, nil
}
