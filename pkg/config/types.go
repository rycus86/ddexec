package config

type Configuration struct {
	Image      string
	Command    []string // TODO simple string
	Name       string
	Dockerfile string
	Privileged bool // TODO not sure if we should support this
	Volumes    []VolumeConfig

	Filename    string `yaml:"-"`
	DesktopMode bool   `yaml:"-"`
}

type VolumeConfig struct {
	Type     string
	Source   string
	Target   string
	ReadOnly bool `yaml:"read_only"`

	// TODO volume options, etc.
}
