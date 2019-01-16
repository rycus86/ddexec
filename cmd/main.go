package main

import (
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/exec"
	"github.com/rycus86/ddexec/pkg/parse"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		println(`
Usage:

	ddexec -d|--desktop <image>		Run an X11 desktop session from the given image
	ddexec <example.dapp.yaml> 		Run an application in the existing session
`)
		os.Exit(1)
	}

	arg := os.Args[1]

	if arg == "-d" || arg == "--desktop" {
		os.Exit(exec.Run(&config.Configuration{
			Image:       os.Args[2],
			Name:        "ddexec-session",
			DesktopMode: true,
		}))
	} else {
		conf := parse.ParseConfiguration(os.Args[1])
		os.Exit(exec.Run(conf))
	}
}
