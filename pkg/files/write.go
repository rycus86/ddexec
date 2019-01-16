package files

import "io/ioutil"

func WriteToTempfile(content string) string {
	target, err := ioutil.TempFile("", "ddexec.*.tmp")
	if err != nil {
		panic(err)
	}
	defer target.Close()

	if _, err := target.WriteString(content); err != nil {
		panic(err)
	}

	return target.Name()
}
