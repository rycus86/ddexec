package exec

import (
	"errors"
	"github.com/rycus86/ddexec/pkg/config"
)

type appWithConfig struct {
	Name   string
	Config *config.AppConfiguration
}

func Sorted(g *config.GlobalConfiguration) []appWithConfig {
	var apps []appWithConfig
	var added = map[string]bool{}

	for len(apps) < len(*g) {
		initialLen := len(apps)

		for name, item := range *g {
			if added[name] {
				continue
			}

			canStart := true
			for _, dep := range item.DependsOn {
				if !added[dep] {
					canStart = false
					break
				}
			}

			if canStart {
				apps = append(apps, appWithConfig{
					Name:   name,
					Config: item,
				})
				added[name] = true
			}
		}

		if initialLen >= len(apps) {
			panic(errors.New("cannot resolve dependencies"))
		}
	}

	return apps
}
