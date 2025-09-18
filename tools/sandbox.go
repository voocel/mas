package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	runtimepkg "github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// ResourceLimits describes sandbox resource limits
type ResourceLimits struct {
	MaxMemory     int64         `json:"max_memory"`     // Maximum memory usage (bytes)
	MaxCPUTime    time.Duration `json:"max_cpu_time"`   // Maximum CPU time
	MaxExecTime   time.Duration `json:"max_exec_time"`  // Maximum execution time
	MaxGoroutines int           `json:"max_goroutines"` // Maximum number of goroutines
}

// DefaultResourceLimits provides default limits
var DefaultResourceLimits = &ResourceLimits{
	MaxMemory:     100 * 1024 * 1024, // 100MB
	MaxCPUTime:    10 * time.Second,
	MaxExecTime:   30 * time.Second,
	MaxGoroutines: 10,
}

// SecurityPolicy defines sandbox security policies
type SecurityPolicy struct {
	AllowNetworkAccess bool     `json:"allow_network_access"`
	AllowFileAccess    bool     `json:"allow_file_access"`
	AllowedDomains     []string `json:"allowed_domains"`
	BlockedDomains     []string `json:"blocked_domains"`
	AllowedPaths       []string `json:"allowed_paths"`
	BlockedPaths       []string `json:"blocked_paths"`
}

// DefaultSecurityPolicy provides default policies
var DefaultSecurityPolicy = &SecurityPolicy{
	AllowNetworkAccess: true,
	AllowFileAccess:    false,
	AllowedDomains:     []string{},
	BlockedDomains:     []string{},
	AllowedPaths:       []string{},
	BlockedPaths:       []string{},
}

// Sandbox isolates tool execution
type Sandbox struct {
	limits  *ResourceLimits
	policy  *SecurityPolicy
	monitor *ResourceMonitor
	enabled bool
	mu      sync.RWMutex
}

// NewSandbox creates a sandbox
func NewSandbox(limits *ResourceLimits, policy *SecurityPolicy) *Sandbox {
	if limits == nil {
		limits = DefaultResourceLimits
	}
	if policy == nil {
		policy = DefaultSecurityPolicy
	}

	return &Sandbox{
		limits:  limits,
		policy:  policy,
		monitor: NewResourceMonitor(),
		enabled: true,
	}
}

// Execute runs the tool within the sandbox
func (s *Sandbox) Execute(tool Tool, ctx runtimepkg.Context, input json.RawMessage) (json.RawMessage, error) {
	if !s.enabled {
		return tool.Execute(ctx, input)
	}

	// Check security policies
	if err := s.checkSecurityPolicy(tool, input); err != nil {
		return nil, schema.NewToolError(tool.Name(), "security_check", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, s.limits.MaxExecTime)
	defer cancel()

	// Start resource monitoring
	s.monitor.Start()
	defer s.monitor.Stop()

	// Execute the tool in a goroutine
	resultChan := make(chan executionResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- executionResult{
					err: fmt.Errorf("tool execution panicked: %v", r),
				}
			}
		}()

		result, err := tool.Execute(ctx, input)
		resultChan <- executionResult{
			result: result,
			err:    err,
		}
	}()

	// Wait for completion or timeout
	select {
	case result := <-resultChan:
		// Inspect resource usage
		if err := s.checkResourceUsage(); err != nil {
			return nil, schema.NewToolError(tool.Name(), "resource_check", err)
		}
		return result.result, result.err

	case <-execCtx.Done():
		return nil, schema.NewToolError(tool.Name(), "execute", schema.ErrToolTimeout)
	}
}

// checkSecurityPolicy validates security policies
func (s *Sandbox) checkSecurityPolicy(tool Tool, input json.RawMessage) error {
	// Perform security checks based on tool type and input
	// For example: validate network or file access

	// Example: verify network access permissions
	if !s.policy.AllowNetworkAccess {
		// Determine whether the tool requires network access
		if isNetworkTool(tool) {
			return schema.ErrToolSandboxViolation
		}
	}

	// Verify that file access is permitted
	if !s.policy.AllowFileAccess {
		if isFileTool(tool) {
			return schema.ErrToolSandboxViolation
		}
	}

	return nil
}

// checkResourceUsage enforces resource limits
func (s *Sandbox) checkResourceUsage() error {
	usage := s.monitor.GetUsage()

	// Check memory usage
	if usage.MemoryUsage > s.limits.MaxMemory {
		return fmt.Errorf("memory usage exceeded limit: %d > %d", usage.MemoryUsage, s.limits.MaxMemory)
	}

	// Check goroutine count
	if usage.GoroutineCount > s.limits.MaxGoroutines {
		return fmt.Errorf("goroutine count exceeded limit: %d > %d", usage.GoroutineCount, s.limits.MaxGoroutines)
	}

	return nil
}

// Enable activates the sandbox
func (s *Sandbox) Enable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = true
}

// Disable deactivates the sandbox
func (s *Sandbox) Disable() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = false
}

// IsEnabled reports whether the sandbox is enabled
func (s *Sandbox) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// executionResult contains execution output
type executionResult struct {
	result json.RawMessage
	err    error
}

// ResourceUsage captures usage metrics
type ResourceUsage struct {
	MemoryUsage    int64
	GoroutineCount int
	StartTime      time.Time
	EndTime        time.Time
}

// ResourceMonitor tracks resource usage
type ResourceMonitor struct {
	startTime       time.Time
	startMemory     int64
	startGoroutines int
	mu              sync.RWMutex
}

// NewResourceMonitor constructs a resource monitor
func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{}
}

// Start begins monitoring
func (m *ResourceMonitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startTime = time.Now()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	m.startMemory = int64(memStats.Alloc)
	m.startGoroutines = runtime.NumGoroutine()
}

// Stop stops monitoring
func (m *ResourceMonitor) Stop() {
	// Place for cleanup if needed
}

// GetUsage returns resource usage
func (m *ResourceMonitor) GetUsage() ResourceUsage {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return ResourceUsage{
		MemoryUsage:    int64(memStats.Alloc) - m.startMemory,
		GoroutineCount: runtime.NumGoroutine() - m.startGoroutines,
		StartTime:      m.startTime,
		EndTime:        time.Now(),
	}
}

// Determine whether the tool performs network operations
func isNetworkTool(tool Tool) bool {
	// Inspect the tool name or type
	name := tool.Name()
	networkTools := []string{"web_search", "http_client", "api_call"}

	for _, nt := range networkTools {
		if name == nt {
			return true
		}
	}
	return false
}

// Determine whether the tool accesses files
func isFileTool(tool Tool) bool {
	// Inspect the tool name or type
	name := tool.Name()
	fileTools := []string{"file_system", "file_read", "file_write"}

	for _, ft := range fileTools {
		if name == ft {
			return true
		}
	}
	return false
}
