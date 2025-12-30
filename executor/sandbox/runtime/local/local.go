package local

import (
	"context"
	"errors"
	"time"

	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/executor/sandbox/policy"
	"github.com/voocel/mas/executor/sandbox/runtime"
	"github.com/voocel/mas/tools"
	"github.com/voocel/mas/tools/builtin"
)

type Runtime struct {
	Registry *tools.Registry
}

var _ runtime.Runtime = (*Runtime)(nil)

func NewRuntime(registry *tools.Registry) *Runtime {
	return &Runtime{Registry: registry}
}

func NewDefaultRuntime() *Runtime {
	registry := tools.NewRegistry()
	registerBuiltinTools(registry)
	return &Runtime{Registry: registry}
}

func (r *Runtime) CreateSandbox(ctx context.Context, req sandbox.CreateSandboxRequest) (*sandbox.CreateSandboxResponse, error) {
	if r == nil {
		return nil, errors.New("runtime is nil")
	}
	id := req.SandboxID
	if id == "" {
		id = "local"
	}
	_ = ctx
	return &sandbox.CreateSandboxResponse{SandboxID: id, Status: sandbox.StatusOK}, nil
}

func (r *Runtime) ExecuteTool(ctx context.Context, req sandbox.ExecuteToolRequest) (*sandbox.ExecuteToolResponse, error) {
	if r == nil {
		return nil, errors.New("runtime is nil")
	}
	registry := r.Registry
	if registry == nil {
		return nil, errors.New("registry is nil")
	}

	if req.Policy.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Policy.Timeout)
		defer cancel()
	}

	tool, ok := registry.Get(req.Tool.Name)
	if !ok {
		return &sandbox.ExecuteToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     sandbox.StatusError,
			Error:      &sandbox.ErrorDetail{Code: sandbox.CodeInvalidRequest, Message: "tool not found"},
			ExitCode:   1,
		}, nil
	}

	if err := policy.ValidateToolPolicy(req.Policy, tool, req.Tool.Args); err != nil {
		return &sandbox.ExecuteToolResponse{
			ToolCallID: req.ToolCallID,
			Status:     sandbox.StatusError,
			Error:      &sandbox.ErrorDetail{Code: sandbox.CodePolicyDenied, Message: err.Error()},
			ExitCode:   1,
		}, nil
	}

	start := time.Now()
	result, err := tool.Execute(ctx, req.Tool.Args)
	resp := &sandbox.ExecuteToolResponse{
		ToolCallID: req.ToolCallID,
		Status:     sandbox.StatusOK,
		Result:     result,
		Usage:      &sandbox.Usage{CPUMs: int(time.Since(start).Milliseconds())},
	}
	if err != nil {
		resp.Status = sandbox.StatusError
		resp.Error = &sandbox.ErrorDetail{Code: sandbox.CodeToolFailed, Message: err.Error()}
		resp.ExitCode = 1
	}
	return resp, nil
}

func (r *Runtime) DestroySandbox(ctx context.Context, req sandbox.DestroySandboxRequest) (*sandbox.DestroySandboxResponse, error) {
	if r == nil {
		return nil, errors.New("runtime is nil")
	}
	_ = ctx
	_ = req
	return &sandbox.DestroySandboxResponse{Status: sandbox.StatusOK}, nil
}

func registerBuiltinTools(registry *tools.Registry) {
	_ = registry.Register(builtin.NewCalculator())
	_ = registry.Register(builtin.NewFileSystemTool(nil, 0))
	_ = registry.Register(builtin.NewHTTPClientTool(0))
	_ = registry.Register(builtin.NewWebSearchTool(""))
	_ = registry.Register(builtin.NewFetchTool(0))
}
