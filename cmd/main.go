package main

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/env"
	"github.com/rycus86/ddexec/pkg/exec"
	"github.com/rycus86/ddexec/pkg/parse"
	"github.com/rycus86/ddexec/pkg/xdgopen"
	"os"
	"strings"
)

func main() {
	checkArgs()
	os.Exit(runMain())
}

func checkArgs() {
	if name, err := os.Executable(); err == nil && strings.HasSuffix(name, "/xdg-open") {
		xdgopen.CheckArgs()
		return
	}

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

DDEXEC_DESKTOP_MODE     Assume launching a new X desktop manager (shares udev)
DDEXEC_PULL             Pull the parent images
DDEXEC_NO_CACHE         Don't use the build cache when building images
DDEXEC_REBUILD          Pull the parent images and don't use the build cache
DDEXEC_IMAGE_ONLY       Exit after building the images
DDEXEC_INTERACTIVE      Attach stdin for interactive sessions
DDEXEC_TTY              Configure the terminal (tty mode)
DDEXEC_HOSTNAMES        Comma-separated, then ':' separated hostname mappings (use 'host' for the bridge gateway)
KEEP_USER               Keep the user in the target image (instead of injecting the host user)
PASSWORD_FILE           Password file to use to generate the container user's password
USE_HOST_X11            Use the X11 socket from the host rather than from a shared volume
USE_HOST_DBUS           Use the DBus sockets from the host rather than from a shared volume
USE_HOST                Use the X11 and DBus sockets from the host
DO_NOT_SHARE_X11        Do not share the X11 socket
DO_NOT_SHARE_DBUS       Do not share the DBus sockets
DO_NOT_SHARE_SHM        Do not share /dev/shm
DO_NOT_SHARE_SOUND      Do not share /dev/snd
DO_NOT_SHARE_VIDEO      Do not share /dev/dri and /dev/video0
DO_NOT_SHARE_DOCKER     Do not share the Docker Engine API socket
DO_NOT_SHARE_HOME       Do not share a common HOME folder with the application
DO_NOT_SHARE_TOOLS      Do not share the ddexec tools with the application
FIX_HOME_ARGS           Fix up the home path in command arguments (replace ${HOME} with ${DDEXEC_HOME})
YUBIKEY_SUPPORT         Enable YubiKey support in the container (requires privileged mode)
DDEXEC_UNIQUE_NAMES 	If you want unique container names with a timestamp instead of a counter
DDEXEC_MAPPING_DIR      Directory to use for storing shared information (xdg-open mappings for example)
DDEXEC_DEBUG            Print debug messages
DDEXEC_TIMER            Print code execution timing information`)

		fmt.Println()
		fmt.Println("ddexec version", config.GetVersion(), "( https://github.com/rycus86/ddexec )")

		os.Exit(0)
	}

	debug.LogTime("checkArgs")
}

func runMain() int {
	if name, err := os.Executable(); err == nil && strings.HasSuffix(name, "/xdg-open") {
		os.Exit(xdgopen.Invoke(os.Args[1]))
	}

	if debug.IsEnabled() {
		fmt.Println("Starting...")
	}

	control.StartServerIfNecessary()

	debug.LogTime("controlServerStarted")

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

		debug.LogTime("runClosers")
	}()

	globalConfig := parse.ParseConfiguration(os.Args[1])

	debug.LogTime("configParsed")

	var exitCode int

	for _, item := range exec.Sorted(globalConfig) {
		code, closer := run(item.Name, item.Config)

		if closer != nil {
			closers = append([]func(){closer}, closers...)
		}

		exitCode = code
	}

	return exitCode
}

func run(name string, configuration *config.AppConfiguration) (int, func()) {
	if debug.IsEnabled() {
		fmt.Println("Starting", name, "...")
	}

	prepareConfiguration(name, configuration)

	debug.LogTime("prepareConfig")

	sc := getStartupConfiguration(configuration)

	debug.LogTime("startupConfig")

	ch, closer := exec.Run(configuration, sc)
	if ch == nil {
		return 0, nil
	}

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
			UseDefaults: true,
		}
	} else {
		c.StartupConfiguration = nil // null it out
	}

	if sc.UseDefaults {
		sc.ShareX11 = setIfUnset(sc.ShareX11, "DO_NOT_SHARE_X11")
		sc.ShareDBus = setIfUnset(sc.ShareDBus, "DO_NOT_SHARE_DBUS")
		sc.ShareShm = setIfUnset(sc.ShareShm, "DO_NOT_SHARE_SHM")
		sc.ShareSound = setIfUnset(sc.ShareSound, "DO_NOT_SHARE_SOUND")
		sc.ShareVideo = setIfUnset(sc.ShareVideo, "DO_NOT_SHARE_VIDEO")
		sc.ShareDockerSocket = setIfUnset(sc.ShareDockerSocket, "DO_NOT_SHARE_DOCKER")
		sc.ShareHomeDir = setIfUnset(sc.ShareHomeDir, "DO_NOT_SHARE_HOME")
		sc.ShareTools = setIfUnset(sc.ShareTools, "DO_NOT_SHARE_TOOLS")
	}

	sc.DesktopMode = sc.DesktopMode || env.IsSet("DDEXEC_DESKTOP_MODE")
	sc.KeepUser = sc.KeepUser || env.IsSet("KEEP_USER")
	sc.UseHostX11 = sc.UseHostX11 || env.IsSet("USE_HOST_X11") || env.IsSet("USE_HOST")
	sc.UseHostDBus = sc.UseHostDBus || env.IsSet("USE_HOST_DBUS") || env.IsSet("USE_HOST")
	sc.FixHomeArgs = sc.FixHomeArgs || env.IsSet("FIX_HOME_ARGS")
	sc.YubiKeySupport = sc.YubiKeySupport || env.IsSet("YUBIKEY_SUPPORT")

	if sc.PasswordFile == "" && env.IsSet("PASSWORD_FILE") {
		sc.PasswordFile = os.Getenv("PASSWORD_FILE")
	}

	if env.IsSet("DDEXEC_HOSTNAMES") {
		sc.Hostnames = append(sc.Hostnames, strings.Split(os.Getenv("DDEXEC_HOSTNAMES"), ",")...)
	}

	sc.XorgLogs = "/var/tmp/ddexec-xorg-logs"

	sc.Args = args

	return sc
}

func setIfUnset(cfg *bool, key string) *bool {
	if cfg != nil {
		return cfg
	} else {
		value := env.IsNotSet(key)
		return &value
	}
}
