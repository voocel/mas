package microvm

import (
	"encoding/json"
	"errors"
	"os"
)

type NetworkConfig struct {
	Enabled      bool     `json:"enabled"`
	TapDevice    string   `json:"tap_device"`
	MacAddress   string   `json:"mac_address"`
	AllowedCIDRs []string `json:"allowed_cidrs"`
}

type PoolConfig struct {
	Size int `json:"size"`
}

type MachineConfig struct {
	VCPUCount int `json:"vcpu_count"`
	MemMiB    int `json:"mem_mib"`
}

type VsockConfig struct {
	CID     uint32 `json:"cid"`
	Port    uint32 `json:"port"`
	UDSPath string `json:"uds_path"`
}

type ToolRunnerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type CgroupConfig struct {
	Path           string `json:"path"`
	CPUQuotaUs     int64  `json:"cpu_quota_us"`
	CPUPeriodUs    int64  `json:"cpu_period_us"`
	CPUWeight      int    `json:"cpu_weight"`
	MemoryMaxBytes int64  `json:"memory_max_bytes"`
	PidsMax        int    `json:"pids_max"`
}

type DriveConfig struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	ReadOnly bool   `json:"read_only"`
}

type Config struct {
	FirecrackerBin string           `json:"firecracker_bin"`
	KernelImage    string           `json:"kernel_image"`
	RootFS         string           `json:"rootfs"`
	Snapshot       string           `json:"snapshot"`
	APISocket      string           `json:"api_socket"`
	LogPath        string           `json:"log_path"`
	MetricsPath    string           `json:"metrics_path"`
	BootArgs       string           `json:"boot_args"`
	Machine        MachineConfig    `json:"machine"`
	VSock          VsockConfig      `json:"vsock"`
	ToolRunner     ToolRunnerConfig `json:"tool_runner"`
	Network        NetworkConfig    `json:"network"`
	Pool           PoolConfig       `json:"pool"`
	Cgroup         CgroupConfig     `json:"cgroup"`
	Drives         []DriveConfig    `json:"drives"`
}

func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, errors.New("config path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
