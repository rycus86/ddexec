package exec

import (
	"fmt"
	"github.com/docker/docker/pkg/term"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/convert"
	"github.com/rycus86/ddexec/pkg/debug"
	"github.com/rycus86/ddexec/pkg/xdgopen"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const DDEXEC_ENV = "DDEXEC_ENV"

func prepareEnvironment(c *config.AppConfiguration, sc *config.StartupConfiguration) []string {
	var env []string

	env = append(env, prepareDdexecEnvironment()...)
	env = append(env, prepareX11Environment(sc)...)
	env = append(env, prepareTimezoneEnvironment()...)
	env = append(env, preparePathEnvironment(sc)...)
	env = append(env, prepareTtySizeEnvironment(c, sc)...)

	if !sc.KeepUser {
		env = append(env, prepareUserEnvironment()...)
	}

	if sc.IsSet(sc.ShareDBus) {
		env = append(env, prepareDBusEnvironment()...)
	}

	env = append(env, convert.ToStringSlice(c.Environment)...)

	if debug.IsEnabled() {
		for _, e := range env {
			fmt.Println("env:", e)
		}
	}

	return env
}

func prepareDdexecEnvironment() []string {
	env := []string{
		DDEXEC_ENV + "=" + strconv.Itoa(1),
		control.EnvHome + "=" + control.GetHostHome(),
		control.EnvServerSocket + "=" + control.GetServerSocket(),
		xdgopen.EnvControlDir + "=" + xdgopen.GetMappingDirectory(),
	}

	if hosts, ok := os.LookupEnv("DDEXEC_HOSTNAMES"); ok {
		env = append(env, fmt.Sprintf("%s=%s", "DDEXEC_HOSTNAMES", hosts))
	}

	if uniqueNames, ok := os.LookupEnv("DDEXEC_UNIQUE_NAMES"); ok {
		env = append(env, fmt.Sprintf("%s=%s", "DDEXEC_UNIQUE_NAMES", uniqueNames))
	}

	return env
}

func prepareX11Environment(sc *config.StartupConfiguration) []string {
	var env []string

	if sc.DesktopMode {
		env = append(env, "XAUTHORITY=/tmp/.server.xauth")
	} else {
		env = append(env, "DISPLAY="+os.Getenv("DISPLAY"))
		env = append(env, "XAUTHORITY="+getXauth())
	}

	return env
}

func prepareTimezoneEnvironment() []string {
	timezone := os.Getenv("TZ")

	if timezone == "" {
		if fi, err := os.Stat("/etc/timezone"); err == nil && !fi.IsDir() && fi.Size() > 0 {
			if tzdata, err := ioutil.ReadFile("/etc/timezone"); err == nil {
				timezone = strings.TrimSpace(string(tzdata))
			}

		} else if zi, err := os.Readlink("/etc/localtime"); err == nil && strings.HasPrefix(zi, "/usr/share/zoneinfo/") {
			timezone = strings.TrimPrefix(zi, "/usr/share/zoneinfo/")
		}
	}

	if timezone != "" {
		return []string{"TZ=" + timezone}
	} else {
		return []string{}
	}
}

func preparePathEnvironment(sc *config.StartupConfiguration) []string {
	if sc.EnvPath != "" {
		sc.EnvPath = "/usr/local/ddexec-xdg/bin:" + sc.EnvPath + ":/usr/local/ddexec/bin"
	} else {
		sc.EnvPath = "/usr/local/ddexec-xdg/bin:/usr/local/ddexec/bin"
	}

	return []string{"PATH=" + sc.EnvPath}
}

func prepareUserEnvironment() []string {
	username := os.Getenv("USER")
	if username == "" {
		username = getUsername()
	}

	return []string{
		"HOME=" + os.Getenv("HOME"),
		"USER=" + username,
	}
}

func prepareDBusEnvironment() []string {
	var env []string

	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		env = append(env, "DBUS_SESSION_BUS_ADDRESS="+os.Getenv("DBUS_SESSION_BUS_ADDRESS"))
	} else if _, err := os.Stat("/run/user/" + strconv.Itoa(os.Getuid()) + "/bus"); err == nil {
		env = append(env, "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/"+strconv.Itoa(os.Getuid())+"/bus")
	}

	if os.Getenv("XDG_RUNTIME_DIR") != "" {
		env = append(env, "XDG_RUNTIME_DIR="+os.Getenv("XDG_RUNTIME_DIR"))
	} else if _, err := os.Stat("/run/user/" + strconv.Itoa(os.Getuid())); err == nil {
		env = append(env, "XDG_RUNTIME_DIR=/run/user/"+strconv.Itoa(os.Getuid()))
	}

	return env
}

func prepareTtySizeEnvironment(c *config.AppConfiguration, sc *config.StartupConfiguration) []string {
	if !c.StdinOpen && !c.Tty {
		return nil
	}

	if !sc.StdOutIsTerminal {
		return nil
	}

	fd, _ := term.GetFdInfo(os.Stdin)

	ws, err := term.GetWinsize(fd)
	if err != nil {
		return nil
	} else if ws.Height == 0 && ws.Width == 0 {
		return nil
	}

	return []string{
		"LINES=" + strconv.Itoa(int(ws.Height)),
		"COLUMS=" + strconv.Itoa(int(ws.Width)),
	}
}
