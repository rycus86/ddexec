package parse

import (
	"github.com/rycus86/ddexec/pkg/config"
	"gopkg.in/yaml.v2"
	"os"
	"path"
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

	replacer := &replacer{Filepath: path}
	replacer.postProcess(c)

	return c
}

type replacer struct {
	Filepath string
}

func (r *replacer) postProcess(c *config.Configuration) {
	c.Command = r.replaceVars(c.Command)
	c.SecurityOpts = r.replaceVars(c.SecurityOpts)
}

func (r *replacer) replaceVars(source []string) []string {
	var target []string

	for _, item := range source {
		target = append(target, os.Expand(item, r.variableMapper))
	}

	return target
}

func (r *replacer) variableMapper(key string) string {
	switch key {
	case "0":
		if exec, err := os.Executable(); err == nil {
			return exec
		}

	case "SOURCE":
		if path.IsAbs(r.Filepath) {
			return r.Filepath
		} else if dir, err := os.Getwd(); err == nil {
			return path.Join(dir, r.Filepath)
		}

	case "SOURCE_DIR":
		if path.IsAbs(r.Filepath) {
			return path.Dir(r.Filepath)
		} else if dir, err := os.Getwd(); err == nil {
			return path.Dir(path.Join(dir, r.Filepath))
		}

	case "PWD":
		if dir, err := os.Getwd(); err == nil {
			return dir
		}

	}

	return key
}
