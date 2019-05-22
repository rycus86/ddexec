package xdgopen

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"os"
)

func CheckArgs() {
	if len(os.Args) != 2 {
		fmt.Println("Error: Expected a file or a URL as the only parameter.")
		fmt.Println("Use `-h` or `--help` for options")
		os.Exit(1)
	}

	if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "--manual" {
		fmt.Println(`xdg-open {file|URL}
xdg-open {--help|--manual|--version}

xdg-open emulation by ddexec ( https://github.com/rycus86/ddexec )

This program opens a file or URL in the user's preferred application
just like the real xdg-open would, but works across ddexec containers.

The mime handlers are configured in the ddexec config yaml:
  x-startup:
    xdg_open:
      text/plain: vim <arg>
      x-scheme-handler/http:  chrome --user-data-dir=/data <arg>
      x-scheme-handler/https: chrome --user-data-dir=/data <arg>

Exit Codes
  An exit code of 0 indicates success while a non-zero exit code indicates failure.
  The following failure codes can be returned:
    1 Error in command line syntax.
    2 One of the files passed on the command line did not exist.
    3 A required tool could not be found.
    4 The action failed.`)
	}

	if os.Args[1] == "-v" || os.Args[1] == "--version" {
		fmt.Println("xdg-open emulation")
		fmt.Println("ddexec version", config.GetVersion(), "( https://github.com/rycus86/ddexec )")
		os.Exit(0)
	}
}
