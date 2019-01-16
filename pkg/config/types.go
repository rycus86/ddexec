package config

type Configuration struct {
	Image      string
	Name       string
	Dockerfile string
	Privileged bool // TODO not sure if we should support this

	Filename    string `yaml:"-"`
	DesktopMode bool   `yaml:"-"`
}
