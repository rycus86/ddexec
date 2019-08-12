package exec

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func getSelfContainerId() string {
	if f, err := os.Open("/proc/self/cgroup"); err != nil {
		if os.IsNotExist(err) {
			return ""
		} else {
			panic(err)
		}
	} else {
		defer f.Close()

		contents, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}

		matcher := regexp.MustCompile(".*/([0-9a-f]{64})$")

		for _, line := range strings.Split(string(contents), "\n") {
			if matcher.MatchString(line) {
				return matcher.ReplaceAllString(line, "$1")
			}
		}

		return ""
	}
}
