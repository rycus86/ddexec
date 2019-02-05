package main

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/env"
	"github.com/rycus86/ddexec/pkg/exec"
	"github.com/rycus86/ddexec/pkg/parse"
	"os"
)

func main() {
	checkArgs()
	os.Exit(runMain())
}

func checkArgs() {
	if debug.IsEnabled() {
		fmt.Println("Args:", os.Args)
	}

	if len(os.Args) < 2 {
		fmt.Println("Error: Expected a configuration file as the first parameter.")
		fmt.Println("Use `-h` or `--help` for options")
		os.Exit(1)
	}

	if os.Args[1] == "-v" || os.Args[1] == "--version" {
		fmt.Println("ddexec version", config.GetVersion(), "( https://github.com/rycus86/ddexec )")
		os.Exit(0)
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Println(`Usage: ./ddexec <config.yml>

Environment variables supported:

DDEXEC_DEBUG            Print debug messages
DDEXEC_DESKTOP_MODE     Assume launching a new X desktop manager (shares udev)
DDEXEC_INTERACTIVE      Attach stdin for interactive sessions
DDEXEC_TTY              Configure the terminal (tty mode)
KEEP_USER               Keep the user in the target image (instead of injecting the host user)
USE_HOST_X11            Use the X11 socket from the host rather than from a shared volume
USE_HOST_DBUS           Use the DBus sockets from the host rather than from a shared volume
DO_NOT_SHARE_X11        Do not share the X11 socket
DO_NOT_SHARE_DBUS       Do not share the DBus sockets
DO_NOT_SHARE_SHM        Do not share /dev/shm
DO_NOT_SHARE_SOUND      Do not share /dev/snd
DO_NOT_SHARE_DOCKER     Do not share the Docker Engine API socket
DO_NOT_SHARE_HOME       Do not share a common HOME folder with the application
DO_NOT_SHARE_TOOLS      Do not share the ddexec tools with the application`)

		os.Exit(0)
	}
}

func runMain() int {
	if debug.IsEnabled() {
		fmt.Println("Starting...")
	}

	control.StartServerIfNecessary()

	var closers []func()
	defer func() {
		if debug.IsEnabled() {
			fmt.Println("Running closers...")
		}

		for _, closer := range closers {
			closer()

			if debug.IsEnabled() {
				fmt.Println("Ran closer")
			}
		}
	}()

	globalConfig := parse.ParseConfiguration(os.Args[1])

	var exitCode int

	for _, item := range exec.Sorted(globalConfig) {
		code, closer := run(item.Name, item.Config)
		closers = append([]func(){closer}, closers...)

		exitCode = code
	}

	return exitCode
}

func run(name string, configuration *config.AppConfiguration) (int, func()) {
	if debug.IsEnabled() {
		fmt.Println("Starting", name, "...")
	}

	prepareConfiguration(name, configuration)

	sc := getStartupConfiguration(configuration)

	ch, closer := exec.Run(configuration, sc)

	if debug.IsEnabled() {
		fmt.Println("Started", name)
	}

	var waitCh = make(chan int, 1)

	go func() {
		if sc.DaemonMode {
			close(waitCh)
		}

		exitCode := <-ch

		if debug.IsEnabled() {
			fmt.Println(name, "has exited with code", exitCode)
		}

		if !sc.DaemonMode {
			waitCh <- exitCode
		}
	}()

	return <-waitCh, closer
}

func prepareConfiguration(name string, c *config.AppConfiguration) {
	if c.Name == "" {
		c.Name = name
	}

	if env.IsSet("DDEXEC_INTERACTIVE") {
		c.StdinOpen = true
	}

	if env.IsSet("DDEXEC_TTY") {
		c.Tty = true
	}
}

func getStartupConfiguration(c *config.AppConfiguration) *config.StartupConfiguration {
	args := os.Args[2:]

	sc := c.StartupConfiguration
	if sc == nil {
		sc = &config.StartupConfiguration{
			ShareX11:          env.IsNotSet("DO_NOT_SHARE_X11"),
			ShareDBus:         env.IsNotSet("DO_NOT_SHARE_DBUS"),
			ShareShm:          env.IsNotSet("DO_NOT_SHARE_SHM"),
			ShareSound:        env.IsNotSet("DO_NOT_SHARE_SOUND"),
			ShareDockerSocket: env.IsNotSet("DO_NOT_SHARE_DOCKER"),
			ShareHomeDir:      env.IsNotSet("DO_NOT_SHARE_HOME"),
			ShareTools:        env.IsNotSet("DO_NOT_SHARE_TOOLS"),
			KeepUser:          env.IsSet("KEEP_USER"),
			UseHostX11:        env.IsSet("USE_HOST_X11"),
			UseHostDBus:       env.IsSet("USE_HOST_DBUS"),
		}
	} else {
		c.StartupConfiguration = nil // null it out
	}

	sc.DesktopMode = sc.DesktopMode || env.IsSet("DDEXEC_DESKTOP_MODE")
	sc.Args = args
	sc.XorgLogs = "/var/tmp/ddexec-xorg-logs"

	return sc
}
