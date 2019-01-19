package debug

import "os"

func IsEnabled() bool {
	return os.Getenv("DDEXEC_DEBUG") != ""
}
