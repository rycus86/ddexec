package debug

import (
	"github.com/rycus86/ddexec/pkg/env"
)

func IsEnabled() bool {
	return env.IsSet("DDEXEC_DEBUG")
}
