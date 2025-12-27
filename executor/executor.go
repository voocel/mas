package executor

import (
	"context"

	"github.com/voocel/mas/schema"
)

// ToolExecutor defines the tool execution abstraction.
type ToolExecutor interface {
	Execute(ctx context.Context, call schema.ToolCall, policy Policy) (schema.ToolResult, error)
	Close() error
}
