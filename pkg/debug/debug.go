package debug

import "os"

var isEnabled bool

func IsEnabled() bool {
	return isEnabled || os.Getenv("DDEXEC_DEBUG") != ""
}

func SetEnabled(enabled bool) {
	isEnabled = enabled
}
