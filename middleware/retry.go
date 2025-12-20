package middleware

import (
	"context"
	"time"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

// RetryMiddleware retries LLM calls.
type RetryMiddleware struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

func (m *RetryMiddleware) HandleLLM(ctx context.Context, state *runner.State, req *llm.Request, next runner.LLMHandler) (*llm.Response, error) {
	if m == nil {
		return next(ctx, req)
	}

	attempts := m.MaxAttempts
	if attempts <= 0 {
		attempts = 2
	}
	baseDelay := m.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 200 * time.Millisecond
	}
	maxDelay := m.MaxDelay
	if maxDelay <= 0 {
		maxDelay = 2 * time.Second
	}
	multiplier := m.Multiplier
	if multiplier <= 0 {
		multiplier = 2
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := next(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !schema.IsRetryable(err) {
			return nil, err
		}
		lastErr = err

		if attempt < attempts {
			delay := baseDelay
			for i := 1; i < attempt; i++ {
				delay = time.Duration(float64(delay) * multiplier)
				if delay > maxDelay {
					delay = maxDelay
					break
				}
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, lastErr
}

var _ runner.LLMMiddleware = (*RetryMiddleware)(nil)
