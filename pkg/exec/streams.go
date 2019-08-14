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
	"time"
)

func checkStreams(sc *config.StartupConfiguration) {
	var isTerminal bool

	_, isTerminal = term.GetFdInfo(os.Stdin)
	sc.StdInIsTerminal = isTerminal

	_, isTerminal = term.GetFdInfo(os.Stdout)
	sc.StdOutIsTerminal = isTerminal

	if debug.IsEnabled() {
		fmt.Println("StdInIsTerminal:", sc.StdInIsTerminal, "StdOutIsTerminal:", sc.StdOutIsTerminal)
	}
}

func setupStreams(cli *client.Client, containerID string, c *config.AppConfiguration, sc *config.StartupConfiguration) func() {
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
	closerFunc = func() {
		resp.Close()
	}

	if debug.IsEnabled() {
		fmt.Println("StdinOpen:", c.StdinOpen, "Tty:", c.Tty)
	}

	if c.StdinOpen && sc.StdInIsTerminal {
		// set raw terminal
		inFd, _ := term.GetFdInfo(os.Stdin)
		state, err := term.SetRawTerminal(inFd)
		if err != nil {
			panic(err)
		}
		// restore raw terminal
		closerFunc = func() {
			resp.Close()
			term.RestoreTerminal(inFd, state)
		}
	}

	// handle output
	go func() {
		if sc.StdInIsTerminal && sc.StdOutIsTerminal {
			io.Copy(os.Stdout, resp.Reader)
		} else {
			stdcopy.StdCopy(os.Stdout, os.Stderr, resp.Conn)
		}

		closerFunc()
	}()

	// handle input
	go func() {
		io.Copy(resp.Conn, os.Stdin)

		if !sc.StdInIsTerminal {
			resp.CloseWrite()
		}
	}()

	return closerFunc
}

func resizeTty(cli *client.Client, containerID string) error {
	fd, _ := term.GetFdInfo(os.Stdin)

	ws, err := term.GetWinsize(fd)
	if err != nil {
		return err
	} else if ws.Height == 0 && ws.Width == 0 {
		return nil
	}

	options := types.ResizeOptions{
		Height: uint(ws.Height),
		Width:  uint(ws.Width),
	}

	if debug.IsEnabled() {
		fmt.Println("Resizing", containerID, "to", options)
	}

	return cli.ContainerResize(context.TODO(), containerID, options)
}

func monitorTtySize(cli *client.Client, containerID string, c *config.AppConfiguration, sc *config.StartupConfiguration) {
	if !c.StdinOpen && !c.Tty {
		return
	}

	if !sc.StdOutIsTerminal {
		return
	}

	if err := resizeTty(cli, containerID); err != nil {
		if debug.IsEnabled() {
			fmt.Println("Failed to (initially) resize", containerID, ":", err)
		}

		go func() {
			var err error
			for retry := 0; retry < 5; retry++ {
				time.Sleep(10 * time.Millisecond)
				if err = resizeTty(cli, containerID); err == nil {
					break
				}
			}

			if err != nil && debug.IsEnabled() {
				fmt.Println("Failed to resize", containerID, ":", err)
			}
		}()
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGWINCH)

	go func() {
		for range sigchan {
			resizeTty(cli, containerID)
		}
	}()
}

func restoreTtySize(cli *client.Client, containerID string) {
	if debug.IsEnabled() {
		fmt.Println("Restoring TTY size for", containerID)
	}

	if err := resizeTty(cli, containerID); err != nil && debug.IsEnabled() {
		fmt.Println("Failed to resize", containerID, ":", err)
	}
}
