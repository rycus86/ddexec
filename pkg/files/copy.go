package files

import (
	"io"
	"io/ioutil"
	"os"
)

func CopyToTempfile(src string) string {
	target, err := ioutil.TempFile("", "ddexec.*.tmp")
	if err != nil {
		panic(err)
	}
	defer target.Close()

	source, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer source.Close()

	if _, err := io.Copy(target, source); err != nil {
		panic(err)
	}

	return target.Name()
}
