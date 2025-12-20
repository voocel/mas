package middleware

import (
	"context"
	"time"

	"github.com/voocel/mas/runner"
)

// TimeoutMiddleware provides timeout control for LLM and tools.
type TimeoutMiddleware struct {
	LLMTimeout  time.Duration
	ToolTimeout time.Duration
}

func (m *TimeoutMiddleware) LLMContext(ctx context.Context, state *runner.State) (context.Context, context.CancelFunc) {
	if m == nil || m.LLMTimeout <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, m.LLMTimeout)
}

func (m *TimeoutMiddleware) ToolContext(ctx context.Context, state *runner.ToolState) (context.Context, context.CancelFunc) {
	if m == nil || m.ToolTimeout <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, m.ToolTimeout)
}

var _ runner.ContextMiddleware = (*TimeoutMiddleware)(nil)
