package middleware

import (
	"context"
	"sync/atomic"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
)

// MetricsSnapshot represents a metrics snapshot.
type MetricsSnapshot struct {
	LLMCalls    int64
	LLMErrors   int64
	ToolCalls   int64
	ToolResults int64
	ToolErrors  int64
	Errors      int64
	LastError   string
}

// MetricsObserver provides simple counters.
type MetricsObserver struct {
	llmCalls    atomic.Int64
	llmErrors   atomic.Int64
	toolCalls   atomic.Int64
	toolResults atomic.Int64
	toolErrors  atomic.Int64
	errors      atomic.Int64
	lastError   atomic.Value
}

func (m *MetricsObserver) OnLLMStart(ctx context.Context, state *runner.State, req *llm.Request) {
	m.llmCalls.Add(1)
}

func (m *MetricsObserver) OnLLMEnd(ctx context.Context, state *runner.State, resp *llm.Response, err error) {
	if err != nil {
		m.llmErrors.Add(1)
	}
}

func (m *MetricsObserver) OnToolCall(ctx context.Context, state *runner.ToolState) {
	m.toolCalls.Add(1)
}

func (m *MetricsObserver) OnToolResult(ctx context.Context, state *runner.ToolState) {
	m.toolResults.Add(1)
	if state != nil && state.Result != nil && state.Result.Error != "" {
		m.toolErrors.Add(1)
	}
}

func (m *MetricsObserver) OnError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	m.errors.Add(1)
	m.lastError.Store(err.Error())
}

// Snapshot returns a metrics snapshot.
func (m *MetricsObserver) Snapshot() MetricsSnapshot {
	last, _ := m.lastError.Load().(string)
	return MetricsSnapshot{
		LLMCalls:    m.llmCalls.Load(),
		LLMErrors:   m.llmErrors.Load(),
		ToolCalls:   m.toolCalls.Load(),
		ToolResults: m.toolResults.Load(),
		ToolErrors:  m.toolErrors.Load(),
		Errors:      m.errors.Load(),
		LastError:   last,
	}
}

var _ runner.Observer = (*MetricsObserver)(nil)
