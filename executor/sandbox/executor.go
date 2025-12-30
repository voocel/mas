package sandbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/schema"
)

type SandboxExecutor struct {
	Client    Client
	SandboxID string
}

func NewSandboxExecutor(c Client) *SandboxExecutor {
	return &SandboxExecutor{Client: c}
}

func (e *SandboxExecutor) Execute(ctx context.Context, call schema.ToolCall, policy executor.Policy) (schema.ToolResult, error) {
	if e == nil || e.Client == nil {
		return schema.ToolResult{ID: call.ID, Error: "sandbox executor is nil"}, errors.New("sandbox executor is nil")
	}
	req := ExecuteToolRequest{
		SandboxID:  e.SandboxID,
		ToolCallID: call.ID,
		Tool: ToolSpec{
			Name: call.Name,
			Args: call.Args,
		},
		Policy: policy,
	}
	resp, err := e.Client.ExecuteTool(ctx, req)
	if err != nil {
		return schema.ToolResult{ID: call.ID, Error: err.Error()}, err
	}
	result := schema.ToolResult{ID: call.ID, Result: resp.Result}
	if resp.Error != nil && resp.Error.Message != "" {
		result.Error = resp.Error.Message
		return result, fmt.Errorf("%w: %s", ErrorFromDetail(resp.Error), resp.Error.Message)
	}
	if resp.Status != "" && resp.Status != StatusOK {
		result.Error = resp.Status
		return result, fmt.Errorf("%w: %s", ErrInternal, resp.Status)
	}
	if resp.ExitCode != 0 {
		return result, fmt.Errorf("%w: exit_code=%d", ErrToolFailed, resp.ExitCode)
	}
	return result, nil
}

func (e *SandboxExecutor) Close() error {
	if e == nil || e.Client == nil {
		return nil
	}
	return e.Client.Close()
}
