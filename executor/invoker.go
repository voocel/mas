package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// ExecutorInvoker runs tool calls via ToolExecutor.
type ExecutorInvoker struct {
	Executor ToolExecutor
	Policy   Policy
}

// Invoke implements tools.Invoker.
func (i *ExecutorInvoker) Invoke(ctx context.Context, registry *tools.Registry, calls []schema.ToolCall) ([]schema.ToolResult, error) {
	if i == nil || i.Executor == nil {
		return nil, fmt.Errorf("executor invoker: executor is nil")
	}

	results := make([]schema.ToolResult, len(calls))
	var firstErr error

	for idx, call := range calls {
		var tool tools.Tool
		if registry != nil {
			var ok bool
			tool, ok = registry.Get(call.Name)
			if !ok {
				err := schema.NewToolError(call.Name, "execute", schema.ErrToolNotFound)
				results[idx] = schema.ToolResult{ID: call.ID, Error: err.Error()}
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			if cfg := getToolConfig(tool); cfg != nil && !cfg.Sandbox {
				result, err := executeLocalToolCall(ctx, tool, call)
				results[idx] = result
				if err != nil && firstErr == nil {
					firstErr = err
				}
				continue
			}
		}

		result, err := i.Executor.Execute(ctx, call, i.Policy)
		results[idx] = result
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return results, firstErr
}

func executeLocalToolCall(ctx context.Context, tool tools.Tool, call schema.ToolCall) (schema.ToolResult, error) {
	if tool == nil {
		return schema.ToolResult{ID: call.ID, Error: "tool is nil"}, fmt.Errorf("tool is nil")
	}

	if baseTool, ok := tool.(*tools.BaseTool); ok {
		if err := baseTool.ValidateInput(call.Args); err != nil {
			return schema.ToolResult{ID: call.ID, Error: err.Error()}, err
		}
	}

	execCtx := ctx
	if cfg := getToolConfig(tool); cfg != nil && cfg.Timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	result, err := tool.Execute(execCtx, call.Args)
	toolResult := schema.ToolResult{ID: call.ID, Result: result}
	if err != nil {
		toolResult.Error = err.Error()
	}
	return toolResult, err
}

func getToolConfig(tool tools.Tool) *tools.ToolConfig {
	if baseTool, ok := tool.(*tools.BaseTool); ok {
		return baseTool.Config()
	}
	type configGetter interface {
		Config() *tools.ToolConfig
	}
	if getter, ok := tool.(configGetter); ok {
		return getter.Config()
	}
	return &tools.ToolConfig{Timeout: 30 * time.Second, Sandbox: true}
}
