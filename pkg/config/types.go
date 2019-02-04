package config

import "time"

type StartupConfiguration struct {
	DesktopMode       bool `yaml:"desktop_mode"`
	KeepUser          bool `yaml:"keep_user"`
	UseHostX11        bool `yaml:"use_host_x11"`
	ShareX11          bool `yaml:"share_x11"`
	UseHostDBus       bool `yaml:"use_host_dbus"`
	ShareDBus         bool `yaml:"share_dbus"`
	ShareShm          bool `yaml:"share_shm"`
	ShareSound        bool `yaml:"share_sound"`
	ShareDockerSocket bool `yaml:"share_docker"`
	ShareHomeDir      bool `yaml:"share_home"`
	ShareTools        bool `yaml:"share_tools"`
	DaemonMode        bool `yaml:"daemon"`

	XorgLogs string `yaml:"-"`

	Args []string `yaml:"-"`

	EnvPath   string `yaml:"-"`
	ImageID   string `yaml:"-"`
	ImageUser string `yaml:"-"`
	ImageHome string `yaml:"-"`
}

type AppConfiguration struct {
	Name        string
	Image       string
	Command     []string // TODO simple string
	Volumes     []VolumeConfig
	Tmpfs       []string       // TODO simple string
	DependsOn   []string       `yaml:"depends_on"`
	StopSignal  string         `yaml:"stop_signal"`
	StopTimeout *time.Duration `yaml:"stop_timeout"`
	WorkingDir  string         `yaml:"working_dir"`
	Environment []string       // TODO map[string]string
	Labels      map[string]string

	Privileged   bool     // TODO not sure if we should support this
	GroupAdd     []string `yaml:"group_add"`
	StdinOpen    bool     `yaml:"stdin_open"`
	Tty          bool
	Devices      []string
	SecurityOpts []string `yaml:"security_opt"`
	CapAdd       []string `yaml:"cap_add"`
	CapDrop      []string `yaml:"cap_drop"`
	Ipc          string
	Pid          string
	NetworkMode  string `yaml:"network_mode"`

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

type GlobalConfiguration map[string]*AppConfiguration

func (mc GlobalConfiguration) Get(name string) *AppConfiguration {
	return mc[name]
}
