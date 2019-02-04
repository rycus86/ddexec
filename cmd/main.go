package main

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/exec"
	"github.com/rycus86/ddexec/pkg/parse"
	flag "github.com/spf13/pflag"
	"os"
)

var (
	desktopMode = flag.Bool("desktop", false, "Run the application(s) in desktop mode")
	interactive = flag.BoolP("interactive", "i", false, "Keep STDIN open even if not attached")
	ttyMode     = flag.BoolP("tty", "t", false, "Allocate a pseudo-TTY")
	hostMode    = flag.Bool("host", false, "Use X11 and DBus from the host")
	keepUser    = flag.Bool("keep-user", false, "Keep the user in the target image")
	debugMode   = flag.Bool("debug", false, "Run in debug mode")
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Need one non-flag argument with the configuration file")
		flag.Usage()
		os.Exit(1)
	}

	debug.SetEnabled(*debugMode)

	os.Exit(runMain())
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

	globalConfig := parse.ParseConfiguration(flag.Arg(0))

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

	if *interactive {
		c.StdinOpen = true
	}

	if *ttyMode {
		c.Tty = true
	}
}

func getStartupConfiguration(c *config.AppConfiguration) *config.StartupConfiguration {
	args := flag.Args()[1:]

	sc := c.StartupConfiguration
	if sc == nil {
		sc = &config.StartupConfiguration{
			ShareX11:          os.Getenv("DO_NOT_SHARE_X11") == "",
			ShareDBus:         os.Getenv("DO_NOT_SHARE_DBUS") == "",
			ShareShm:          os.Getenv("DO_NOT_SHARE_SHM") == "",
			ShareSound:        os.Getenv("DO_NOT_SHARE_SOUND") == "",
			ShareDockerSocket: os.Getenv("DO_NOT_SHARE_DOCKER") == "",
			ShareHomeDir:      os.Getenv("DO_NOT_SHARE_HOME") == "",
			ShareTools:        os.Getenv("DO_NOT_SHARE_TOOLS") == "",
			KeepUser:          *keepUser,
			UseHostX11:        *hostMode,
			UseHostDBus:       *hostMode,
		}
	} else {
		c.StartupConfiguration = nil // null it out
	}

	sc.DesktopMode = sc.DesktopMode || *desktopMode
	sc.Args = args
	sc.XorgLogs = "/var/tmp/ddexec-xorg-logs"

	return sc
}
