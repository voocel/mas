//go:build !linux

package microvm

import (
	"context"
	"errors"

	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/executor/sandbox/runtime"
)

type Runtime struct {
	Config Config
}

var _ runtime.Runtime = (*Runtime)(nil)

func NewRuntime(cfg Config) *Runtime {
	return &Runtime{Config: cfg}
}

func (r *Runtime) CreateSandbox(ctx context.Context, req sandbox.CreateSandboxRequest) (*sandbox.CreateSandboxResponse, error) {
	_ = ctx
	_ = req
	return nil, errors.New("microvm runtime is only supported on linux")
}

func (r *Runtime) ExecuteTool(ctx context.Context, req sandbox.ExecuteToolRequest) (*sandbox.ExecuteToolResponse, error) {
	_ = ctx
	_ = req
	return nil, errors.New("microvm runtime is only supported on linux")
}

func (r *Runtime) DestroySandbox(ctx context.Context, req sandbox.DestroySandboxRequest) (*sandbox.DestroySandboxResponse, error) {
	_ = ctx
	_ = req
	return nil, errors.New("microvm runtime is only supported on linux")
}
