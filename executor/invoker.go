package executor

import (
	"context"
	"fmt"

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
		if registry != nil {
			if _, ok := registry.Get(call.Name); !ok {
				err := schema.NewToolError(call.Name, "execute", schema.ErrToolNotFound)
				results[idx] = schema.ToolResult{ID: call.ID, Error: err.Error()}
				if firstErr == nil {
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
