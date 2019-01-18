package exec

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/files"
	"os"
	"os/user"
	"strconv"
	"strings"
)

func prepareUserAndGroupFiles(sc *config.StartupConfiguration) *files.PasswdFiles {
	var passwd, group string
	var temporary bool

	if sc.DesktopMode {
		group = files.CopyToTempfile("/etc/group")
		passwd = files.CopyToTempfile("/etc/passwd")
		files.ModifyFile(passwd, "(?m)^("+getUsername()+":.+:)[^:]*$", "$1/bin/sh")
		temporary = true
	} else {
		passwd = "/etc/passwd"
		group = "/etc/group"
		temporary = false
	}

	shadow := files.WriteToTempfile(strings.TrimSpace(fmt.Sprintf(`
%s:!::0:99999:7:::
root:!::0:99999:7:::
`, getUsername())))

	return &files.PasswdFiles{
		Passwd:    passwd,
		Group:     group,
		Shadow:    shadow,
		Temporary: temporary,
	}
}

func getUserAndGroup() string {
	return strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
}

func getUsername() string {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	return u.Username
}
