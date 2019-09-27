package config

import "time"

type StartupConfiguration struct {
	UseDefaults bool `yaml:"use_defaults"`

	// these are true by default
	ShareX11          *bool `yaml:"share_x11"`
	ShareDBus         *bool `yaml:"share_dbus"`
	ShareShm          *bool `yaml:"share_shm"`
	ShareSound        *bool `yaml:"share_sound"`
	ShareVideo        *bool `yaml:"share_video"`
	ShareDockerSocket *bool `yaml:"share_docker"`
	ShareHomeDir      *bool `yaml:"share_home"`
	ShareTools        *bool `yaml:"share_tools"`

	DesktopMode    bool `yaml:"desktop_mode"`
	KeepUser       bool `yaml:"keep_user"`
	UseHostX11     bool `yaml:"use_host_x11"`
	UseHostDBus    bool `yaml:"use_host_dbus"`
	FixHomeArgs    bool `yaml:"fix_home_args"`
	YubiKeySupport bool `yaml:"yubikey_support"`
	DaemonMode     bool `yaml:"daemon"`

	PasswordFile string `yaml:"password_file"`

	Hostnames       []string          `yaml:"hostnames"`
	XdgOpenMappings map[string]string `yaml:"xdg_open"`

	XorgLogs string `yaml:"-"`

	Args []string `yaml:"-"`

	EnvPath   string `yaml:"-"`
	ImageID   string `yaml:"-"`
	ImageUser string `yaml:"-"`
	ImageHome string `yaml:"-"`

	DaemonHasSeccompSupport bool `yaml:"-"`
	StdInIsTerminal         bool `yaml:"-"`
	StdOutIsTerminal        bool `yaml:"-"`
}

func (sc *StartupConfiguration) IsSet(cfg *bool) bool {
	return cfg != nil && *cfg
}

type AppConfiguration struct {
	Name        string
	Image       string
	Command     interface{}
	Volumes     []interface{}
	Tmpfs       interface{}
	DependsOn   []string       `yaml:"depends_on"`
	StopSignal  string         `yaml:"stop_signal"`
	StopTimeout *time.Duration `yaml:"stop_timeout"`
	WorkingDir  string         `yaml:"working_dir"`
	Environment interface{}
	Labels      map[string]string
	Ports       []string

	ReadOnly     bool `yaml:"read_only"`
	Privileged   bool // TODO not sure if we should support this
	Init         *bool
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

	MemoryLimit       string `yaml:"mem_limit"`
	MemoryReservation string `yaml:"mem_reservation"`
	MemorySwap        string `yaml:"memswap_limit"`
	MemorySwappiness  *int64 `yaml:"mem_swappiness"`
	ShmSize           string `yaml:"shm_size"`
	Cpus              string `yaml:"cpus"`
	CpuShares         int64  `yaml:"cpu_shares"`
	CpuQuota          int64  `yaml:"cpu_quota"`
	CpuPeriod         int64  `yaml:"cpu_period"`
	CpusetCpus        string `yaml:"cpuset"`
	OomScoreAdj       int    `yaml:"oom_score_adj"`
	OomKillDisable    *bool  `yaml:"oom_kill_disable"`
	PidsLimit         int64  `yaml:"pids_limit"`

	Dockerfile string

	StartupConfiguration *StartupConfiguration `yaml:"x-startup"`
}

type GlobalConfiguration map[string]*AppConfiguration
