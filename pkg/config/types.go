package config

type StartupConfiguration struct {
	DesktopMode bool `yaml:"-"`

	KeepUser          bool `yaml:"keep_user"`
	UseHostX11        bool `yaml:"use_host_x11"`
	ShareX11          bool `yaml:"share_x11"`
	UseHostDBus       bool `yaml:"use_host_dbus"`
	ShareDBus         bool `yaml:"share_dbus"`
	ShareShm          bool `yaml:"share_shm"`
	ShareDockerSocket bool `yaml:"share_docker"`
	ShareHomeDir      bool `yaml:"share_home"`
	ShareTools        bool `yaml:"share_tools"`

	XorgLogs string `yaml:"-"`

	Args     []string `yaml:"-"`
	Filename string   `yaml:"-"`

	EnvPath   string `yaml:"-"`
	ImageID   string `yaml:"-"`
	ImageUser string `yaml:"-"`
	ImageHome string `yaml:"-"`
}

type Configuration struct {
	Name    string
	Image   string
	Command []string // TODO simple string
	Volumes []VolumeConfig

	Privileged   bool // TODO not sure if we should support this
	StdinOpen    bool `yaml:"stdin_open"`
	Tty          bool
	Devices      []string
	SecurityOpts []string `yaml:"security_opt"`
	CapAdd       []string `yaml:"cap_add"`
	CapDrop      []string `yaml:"cap_drop"`
	Ipc          string

	MemLimit string `yaml:"mem_limit"`

	Dockerfile string

	StartupConfiguration *StartupConfiguration `yaml:"x-startup"`
}

type VolumeConfig struct {
	Type     string
	Source   string
	Target   string
	ReadOnly bool `yaml:"read_only"`

	// TODO volume options, etc.
}
