package xdgopen

import (
	"github.com/rycus86/ddexec/pkg/control"
	"github.com/rycus86/ddexec/pkg/env"
	"os"
)

const EnvControlDir = "DDEXEC_MAPPING_DIR"

func GetMappingDirectory() string {
	if env.IsSet(EnvControlDir) {
		return os.Getenv(EnvControlDir)
	} else {
		return control.GetDirectoryToShare()
	}
}
