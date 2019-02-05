package control

import (
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/env"
	"os"
	"os/user"
)

const EnvHome = "DDEXEC_HOME"

func GetHostHome() string {
	if env.IsSet(EnvHome) {
		return os.Getenv(EnvHome)
	} else {
		return os.ExpandEnv("${HOME}/.ddexec/home")
	}
}

func getContainerHome(sc *config.StartupConfiguration) string {
	// TODO perhaps read this from /etc/passwd

	if sc.KeepUser {
		home := sc.ImageHome

		if home != "" {
			return home
		} else {
			return "/home/" + sc.ImageUser
		}
	} else {
		return "/home/" + getUsername()
	}
}

func getUsername() string { // TODO duplicate
	if username := os.Getenv("USER"); username != "" {
		return username
	}

	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return u.Username
}

func getTargetUsername(sc *config.StartupConfiguration) string {
	if sc.KeepUser {
		return sc.ImageUser
	} else {
		return getUsername()
	}
}
