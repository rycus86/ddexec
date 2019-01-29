package exec

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/pkg/term"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"io"
	"os"
	"os/signal"
	"syscall"
)

func setupStreams(cli *client.Client, containerID string, c *config.AppConfiguration) func() {
	var closerFunc func()

	resp, err := cli.ContainerAttach(context.TODO(), containerID, types.ContainerAttachOptions{
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		panic(err)
	}

	if debug.IsEnabled() {
		fmt.Println("StdinOpen:", c.StdinOpen, "Tty:", c.Tty)
	}

	if c.StdinOpen {
		// set raw terminal
		inFd, _ := term.GetFdInfo(os.Stdin)
		state, err := term.SetRawTerminal(inFd)
		if err != nil {
			panic(err)
		}
		// restore raw terminal
		closerFunc = func() {
			term.RestoreTerminal(inFd, state)
		}
	}

	// handle output
	go func() {
		if c.Tty {
			io.Copy(os.Stdout, resp.Reader)
		} else {
			stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Reader)
		}
	}()

	// handle input
	go func() {
		io.Copy(resp.Conn, os.Stdin)
	}()

	return closerFunc
}

func monitorTtySize(cli *client.Client, containerID string, c *config.AppConfiguration) {
	if !c.StdinOpen && !c.Tty {
		return
	}

	fd, _ := term.GetFdInfo(os.Stdin)

	resizeTty := func() {
		ws, err := term.GetWinsize(fd)
		if err != nil || (ws.Height == 0 && ws.Width == 0) {
			return
		}

		options := types.ResizeOptions{
			Height: uint(ws.Height),
			Width:  uint(ws.Width),
		}

		cli.ContainerResize(context.TODO(), containerID, options)
	}

	resizeTty()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGWINCH)

	go func() {
		for range sigchan {
			resizeTty()
		}
	}()
}
