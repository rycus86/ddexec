package files

import (
	"io/ioutil"
	"regexp"
)

func ModifyFile(path, pattern, replacement string) {
	regex := regexp.MustCompile(pattern)

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	replaced := regex.ReplaceAll(contents, []byte(replacement))

	if err := ioutil.WriteFile(path, replaced, 0); err != nil {
		panic(err)
	}
}
