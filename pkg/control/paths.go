package control

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/env"
	"os"
	"path/filepath"
	"strings"
)

func Source(path string) string {
	return filepath.Clean(os.Expand(strings.Replace(path, "~", "${HOME}", 1), func(key string) string {
		if debug.IsEnabled() {
			fmt.Println("Source: Looking for", key)
		}

		switch key {
		case "HOME":
			return GetHostHome()
		default:
			return os.Getenv(key)
		}
	}))
}

func EnsureSourceExists(path string) string {
	source := Source(path)
	if strings.HasPrefix(source, GetHostHome()) || strings.HasPrefix(source, "/tmp/") {
		return UnsafeEnsureSourceExists(path)
	} else {
		panic(errors.New("not allowed to use " + path + " as a source for a bind mount"))
	}
}

func UnsafeEnsureSourceExists(path string) string {
	if env.IsSet(EnvServerSocket) {
		if debug.IsEnabled() {
			fmt.Println("Asking control to create", path, "...")
		}

		if p, err := MkdirAll(path); err != nil {
			if debug.IsEnabled() {
				fmt.Println("  failed:", err)
			}
		} else {
			path = p
		}
	} else {
		path = Source(path)

		if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
			if debug.IsEnabled() {
				fmt.Println("Creating directory:", path)
			}

			os.MkdirAll(path, 0777)
		}
	}

	return path
}

func Target(path string, sc *config.StartupConfiguration) string {
	return os.Expand(strings.Replace(path, "~", "${HOME}", 1), func(key string) string {
		if debug.IsEnabled() {
			fmt.Println("Source: Looking for", key)
		}

		switch key {
		case "HOME":
			return getContainerHome(sc)
		case "USER":
			return getTargetUsername(sc)
		default:
			return os.Getenv(key)
		}
	})
}
