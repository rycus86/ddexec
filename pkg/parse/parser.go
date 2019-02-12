package parse

import (
	"bytes"
	"fmt"
	"github.com/rycus86/ddexec/pkg/config"
	"github.com/rycus86/ddexec/pkg/debug"
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
var (
	reVariableWithDefault = regexp.MustCompile("(.+):-(.*)")

	variablesToKeep = []string{"USER"}                  // we'll process these later
	nonInterpolated = []string{"volumes", "dockerfile"} // we shouldn't interpolate these (just yet)
)

func isVariableKept(v string) bool {
	for _, x := range variablesToKeep {
		if x == v {
			return true
		}
	}
	return false
}

func isNotInterpolated(field string) bool {
	for _, x := range nonInterpolated {
		if x == field {
			return true
		}
	}
	return false
}

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

	mapper := newMapper(filepath)

	var rawYaml map[interface{}]interface{}
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&rawYaml); err != nil {
		panic(err)
	}

	var processedYaml = postProcess(rawYaml, mapper)

	if debug.IsEnabled() {
		fmt.Printf("Processed YAML:\n%+v\n", processedYaml)
	}

	var (
		yamlContents = new(bytes.Buffer)
		encoder      = yaml.NewEncoder(yamlContents)
	)
	if err := encoder.Encode(processedYaml); err != nil {
		panic(err)
	}
	encoder.Close()

	c := config.GlobalConfiguration{}

	decoder := yaml.NewDecoder(strings.NewReader(yamlContents.String()))
	decoder.SetStrict(true)

	if err := decoder.Decode(&c); err != nil {
		panic(err)
	}

	return &c
}

func newMapper(filepath string) func(key string) string {
	return func(key string) string {
		if isVariableKept(key) {
			return "${" + key + "}"
		}

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
}

func postProcess(v interface{}, mapping func(string) string) interface{} {
	if m, ok := v.(map[interface{}]interface{}); ok {
		for key, value := range m {
			if isNotInterpolated(key.(string)) {
				continue
			}

			m[key] = postProcess(value, mapping)
		}
	} else if arr, ok := v.([]interface{}); ok {
		for idx, value := range arr {
			arr[idx] = postProcess(value, mapping)
		}
	} else if s, ok := v.(string); ok {
		return os.Expand(s, mapping)
	}

	return v
}
