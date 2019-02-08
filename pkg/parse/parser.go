package parse

import (
	"github.com/rycus86/ddexec/pkg/config"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
)

// TODO other variations:
// https://docs.docker.com/compose/compose-file/compose-file-v2/#variable-substitution
//   ${VARIABLE:-default} evaluates to default if VARIABLE is unset or empty in the environment.
//   ${VARIABLE-default}  evaluates to default only if VARIABLE is unset in the environment.
//   ${VARIABLE:?err}   exits with an error message containing err if VARIABLE is unset or empty in the environment.
//   ${VARIABLE?err}    exits with an error message containing err if VARIABLE is unset in the environment.
var reVariableWithDefault = regexp.MustCompile("(.+):-(.*)")

func ParseConfiguration(filepath string) *config.GlobalConfiguration {
	f, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}

	mapper := func(key string) string {
		switch key {
		case "0":
			if exec, err := os.Executable(); err == nil {
				return exec
			}

		case "SOURCE":
			if path.IsAbs(filepath) {
				return filepath
			} else if dir, err := os.Getwd(); err == nil {
				return path.Join(dir, filepath)
			}

		case "SOURCE_DIR":
			if path.IsAbs(filepath) {
				return path.Dir(filepath)
			} else if dir, err := os.Getwd(); err == nil {
				return path.Dir(path.Join(dir, filepath))
			}

		case "PWD":
			if dir, err := os.Getwd(); err == nil {
				return dir
			}

		case "HOME": // TODO a bit hacky here?
			return "${HOME}" // this is dealt with later

		case "USER": // TODO a bit hacky here?
			return "${USER}" // this is dealt with later

		default:
			if reVariableWithDefault.MatchString(key) {
				if val := os.Getenv(reVariableWithDefault.ReplaceAllString(key, "$1")); val != "" {
					return val
				} else {
					return reVariableWithDefault.ReplaceAllString(key, "$2")
				}
			} else {
				return os.ExpandEnv("${" + key + "}")
			}
		}

		return key
	}

	// TODO maybe parse into map[?]?{} then rewrite the string fields,
	//  then load it into the final struct with mapstructure
	yamlContents := os.Expand(string(data), mapper)

	c := config.GlobalConfiguration{}

	decoder := yaml.NewDecoder(strings.NewReader(yamlContents))
	decoder.SetStrict(true)

	if err := decoder.Decode(&c); err != nil {
		panic(err)
	}

	return &c
}
