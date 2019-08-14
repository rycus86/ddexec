package debug

import (
	"fmt"
	"github.com/rycus86/ddexec/pkg/env"
	"time"
)

var lastTimer = time.Now()

func IsEnabled() bool {
	return env.IsSet("DDEXEC_DEBUG")
}

func IsTimerEnabled() bool {
	return env.IsSet("DDEXEC_TIMER")
}

func LogTime(s string) {
	if !IsTimerEnabled() {
		return
	}

	if s != "" {
		tenthMills := time.Since(lastTimer).Nanoseconds() / int64(100*time.Microsecond)
		fmt.Printf("time :: %25s :: %4d.%1d ms\n", s, tenthMills/10, tenthMills%10)
	}

	lastTimer = time.Now()
}
