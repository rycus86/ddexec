package config

type StartupConfiguration struct {
	DesktopMode       bool
	KeepUser          bool
	ShareX11          bool
	ShareDBus         bool
	ShareDockerSocket bool
	UseHostX11        bool
	SharedHomeDir     bool
	SharedTools       bool

	XorgLogs string

	Filename string

	EnvPath   string
	ImageID   string
	ImageUser string
	ImageHome string
}

type Configuration struct {
	Image      string
	Command    []string // TODO simple string
	Name       string
	Dockerfile string
	Privileged bool // TODO not sure if we should support this
	Volumes    []VolumeConfig
}

type VolumeConfig struct {
	Type     string
	Source   string
	Target   string
	ReadOnly bool `yaml:"read_only"`

	// TODO volume options, etc.
}
