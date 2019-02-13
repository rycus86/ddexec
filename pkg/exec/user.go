package exec

import (
	"bytes"
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/files"
	"github.com/tredoe/osutil/user/crypt"
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	"io/ioutil"
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

	userPasswd := "!"

	if sc.PasswordFile != "" {
		userPasswd = generateHashedPassword(sc.PasswordFile)
	}

	shadow := files.WriteToTempfile(strings.TrimSpace(fmt.Sprintf(`
%s:%s::0:99999:7:::
root:!::0:99999:7:::
`, getUsername(), userPasswd)))

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

func generateHashedPassword(passwordFile string) string {
	passwd, err := ioutil.ReadFile(passwordFile)
	if err != nil {
		panic(err)
	}

	passwd = bytes.Trim(passwd, "\n")

	if strings.HasPrefix(string(passwd), sha512_crypt.MagicPrefix) {
		// already an encoded password (with mkpasswd perhaps)
		return string(passwd)
	}

	sha512 := crypt.New(crypt.SHA512)
	shaSalt := sha512_crypt.GetSalt()
	salt := shaSalt.Generate(16)

	if generated, err := sha512.Generate(passwd, salt); err != nil {
		panic(err)
	} else {
		return generated
	}
}
