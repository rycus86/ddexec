package xdgopen

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/xdgexec"
	"io/ioutil"
	"mime"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Invoke(arg string) int {
	if strings.Contains(arg, "://") {
		parsed, err := url.Parse(arg)
		if err != nil {
			fmt.Println("Invalid URL:", arg, "-", err)
			return 1 // Error in command line syntax.
		}

		return dispatch("x-scheme-handler/"+parsed.Scheme, arg)

	} else {
		if _, err := os.Stat(arg); err != nil && os.IsNotExist(err) {
			fmt.Println("File does not exist:", arg, "-", err)
			return 2 // One of the files passed on the command line did not exist.
		}

		mimetype := mime.TypeByExtension(path.Ext(arg))
		if strings.Contains(mimetype, ";") {
			mimetype = strings.Split(mimetype, ";")[0]
		}

		if mimetype == "" {
			fmt.Println("Mimetype is unknown for", arg)
			return 4 // The action failed.
		}

		return dispatch(mimetype, arg)
	}
}

func dispatch(mimetype string, arg string) int {
	files, err := filepath.Glob(filepath.Join(control.GetDirectoryToShare(), "xdg_open.*"))
	if err != nil {
		fmt.Println("Failed to read xdg-open mappings:", err)
		return 4 // The action failed.
	}

	var targetContainer, mappedCommand string

	for _, filename := range files {
		if contents, err := ioutil.ReadFile(filename); err != nil {
			continue // Failed to read file - TODO maybe log it
		} else {
			for _, line := range strings.Split(string(contents), "\n") {
				if strings.HasPrefix(line, mimetype+"=") {
					mappedCommand = strings.SplitN(line, "=", 2)[1]
					targetContainer = strings.Replace(filepath.Base(filename), "xdg_open.", "", 1)
					break
				}
			}
		}

		if mappedCommand != "" {
			break
		}
	}

	if targetContainer != "" {
		finalCommand := strings.Replace(mappedCommand, "<arg>", arg, -1)

		if ok, err := xdgexec.ExecInContainer(targetContainer, finalCommand); err != nil {
			if ok, err := control.RunCommand(targetContainer, finalCommand); err == nil && ok {
				return 0 // Successful.
			} else {
				return 4 // The action failed.
			}
		} else if ok {
			return 0 // Successful.
		} else {
			return 4 // The action failed.
		}
	}

	fmt.Printf("No tool found for %s (type: %s)\n", arg, mimetype)
	return 3 // A required tool could not be found.
}
