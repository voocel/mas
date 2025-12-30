package sandbox

import (
	"context"
	"encoding/json"

	"github.com/voocel/mas/executor"
)

type Client interface {
	CreateSandbox(ctx context.Context, req CreateSandboxRequest) (*CreateSandboxResponse, error)
	ExecuteTool(ctx context.Context, req ExecuteToolRequest) (*ExecuteToolResponse, error)
	DestroySandbox(ctx context.Context, req DestroySandboxRequest) (*DestroySandboxResponse, error)
	Health(ctx context.Context) (*HealthResponse, error)
	Close() error
}

const (
	StatusOK    = "ok"
	StatusError = "error"
)

type TraceContext struct {
	RunID  string `json:"run_id,omitempty"`
	StepID string `json:"step_id,omitempty"`
}

type ResourceLimits struct {
	CPU       int `json:"cpu,omitempty"`
	MemMB     int `json:"mem_mb,omitempty"`
	DiskMB    int `json:"disk_mb,omitempty"`
	TimeoutMS int `json:"timeout_ms,omitempty"`
}

type CreateSandboxRequest struct {
	SandboxID string          `json:"sandbox_id,omitempty"`
	Policy    executor.Policy `json:"policy"`
	Resources ResourceLimits  `json:"resources,omitempty"`
}

type CreateSandboxResponse struct {
	SandboxID string `json:"sandbox_id"`
	Status    string `json:"status"`
}

type DestroySandboxRequest struct {
	SandboxID string `json:"sandbox_id"`
}

type DestroySandboxResponse struct {
	Status string `json:"status"`
}

type ToolSpec struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}

type ExecuteToolRequest struct {
	SandboxID  string          `json:"sandbox_id,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Tool       ToolSpec        `json:"tool"`
	Policy     executor.Policy `json:"policy,omitempty"`
	Trace      TraceContext    `json:"trace,omitempty"`
}

type ExecuteToolResponse struct {
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Status     string          `json:"status"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      *ErrorDetail    `json:"error,omitempty"`
	ExitCode   int             `json:"exit_code,omitempty"`
	Usage      *Usage          `json:"usage,omitempty"`
	Stderr     string          `json:"stderr,omitempty"`
}

type Usage struct {
	CPUMs int `json:"cpu_ms,omitempty"`
	MemMB int `json:"mem_mb,omitempty"`
}

type ErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}
