package executor

import (
	"time"

	"github.com/voocel/mas/tools"
)

// NetworkPolicy defines network access policy.
type NetworkPolicy struct {
	Enabled      bool
	AllowedHosts []string
}

// Policy defines executor policy.
type Policy struct {
	AllowedTools  []string
	AllowedCaps   []tools.Capability
	Workdir       string
	AllowedPaths  []string
	EnvWhitelist  []string
	Network       NetworkPolicy
	Timeout       time.Duration
	MemoryLimit   uint64
	MaxConcurrent int
}
