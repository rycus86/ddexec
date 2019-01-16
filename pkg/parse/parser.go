package parse

import (
	"github.com/rycus86/ddexec/pkg/config"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

func ParseConfiguration(path string) *config.Configuration {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c := new(config.Configuration)
	decoder := yaml.NewDecoder(f)

	if err := decoder.Decode(c); err != nil {
		panic(err)
	}

	c.Filename = filepath.Base(path)

	var data string
	if err := decoder.Decode(&data); err != nil {
		panic(err)
	} else {
		c.Dockerfile = data
	}

	return c
}
