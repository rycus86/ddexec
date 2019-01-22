package main

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/exec"
	"github.com/rycus86/ddexec/pkg/parse"
	"os"
	"path/filepath"
)

func main() {
	if debug.IsEnabled() {
		fmt.Println("Starting...")
	}

	if len(os.Args) < 2 {
		println(`
Usage:

	ddexec -d|--desktop <image>		Run an X11 desktop session from the given image
	ddexec <example.dapp.yaml> 		Run an application in the existing session
`)
		os.Exit(1)
	}

	conf := getConfiguration()
	startupConf := getStartupConfiguration(conf)

	os.Exit(exec.Run(conf, startupConf))
}

func getConfiguration() *config.Configuration {
	if isDesktopMode() {
		return &config.Configuration{
			Image: os.Args[2],
			Name:  "ddexec-session",
		}
	} else {
		return parse.ParseConfiguration(os.Args[1])
	}
}

func getStartupConfiguration(c *config.Configuration) *config.StartupConfiguration {
	var filename string
	if !isDesktopMode() {
		filename = filepath.Base(os.Args[1])
	}

	var args []string
	if isDesktopMode() {
		args = os.Args[3:]
	} else {
		args = os.Args[2:]
	}

	sc := c.StartupConfiguration
	if sc == nil {
		sc = &config.StartupConfiguration{
			ShareX11:          os.Getenv("DO_NOT_SHARE_X11") == "",
			ShareDBus:         os.Getenv("DO_NOT_SHARE_DBUS") == "",
			ShareShm:          os.Getenv("DO_NOT_SHARE_SHM") == "",
			ShareDockerSocket: os.Getenv("DO_NOT_SHARE_DOCKER") == "",
			ShareHomeDir:      os.Getenv("DO_NOT_SHARE_HOME") == "",
			ShareTools:        os.Getenv("DO_NOT_SHARE_TOOLS") == "",
			KeepUser:          os.Getenv("KEEP_USER") != "",
			UseHostX11:        os.Getenv("USE_HOST_X11") != "",
		}
	} else {
		c.StartupConfiguration = nil // null it out
	}

	sc.Filename = filename
	sc.DesktopMode = isDesktopMode()
	sc.Args = args
	sc.XorgLogs = "/var/tmp/ddexec-xorg-logs"

	return sc
}

func isDesktopMode() bool {
	return os.Args[1] == "-d" || os.Args[1] == "--desktop"
}
