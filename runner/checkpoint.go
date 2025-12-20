package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/voocel/mas/schema"
)

type Checkpoint struct {
	RunID       schema.RunID
	Turn        int
	Input       schema.Message
	Messages    []schema.Message
	ToolCalls   []schema.ToolCall
	ToolResults []schema.ToolResult
	UpdatedAt   time.Time
}

type Checkpointer interface {
	Save(ctx context.Context, checkpoint Checkpoint) error
	Load(ctx context.Context, runID schema.RunID) (*Checkpoint, error)
}

type MemoryCheckpointer struct {
	mu    sync.RWMutex
	store map[schema.RunID]*Checkpoint
}

func NewMemoryCheckpointer() *MemoryCheckpointer {
	return &MemoryCheckpointer{
		store: make(map[schema.RunID]*Checkpoint),
	}
}

func (m *MemoryCheckpointer) Save(_ context.Context, checkpoint Checkpoint) error {
	if checkpoint.RunID == "" {
		return fmt.Errorf("checkpoint: run id is empty")
	}
	cp := checkpoint.Clone()
	m.mu.Lock()
	m.store[checkpoint.RunID] = cp
	m.mu.Unlock()
	return nil
}

func (m *MemoryCheckpointer) Load(_ context.Context, runID schema.RunID) (*Checkpoint, error) {
	if runID == "" {
		return nil, fmt.Errorf("checkpoint: run id is empty")
	}
	m.mu.RLock()
	cp, ok := m.store[runID]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("checkpoint: not found")
	}
	return cp.Clone(), nil
}

func (c *Checkpoint) Clone() *Checkpoint {
	if c == nil {
		return nil
	}
	return &Checkpoint{
		RunID:       c.RunID,
		Turn:        c.Turn,
		Input:       *c.Input.Clone(),
		Messages:    cloneMessages(c.Messages),
		ToolCalls:   cloneToolCalls(c.ToolCalls),
		ToolResults: cloneToolResults(c.ToolResults),
		UpdatedAt:   c.UpdatedAt,
	}
}

func cloneMessages(messages []schema.Message) []schema.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]schema.Message, len(messages))
	for i, msg := range messages {
		out[i] = *msg.Clone()
	}
	return out
}

func cloneToolCalls(calls []schema.ToolCall) []schema.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]schema.ToolCall, len(calls))
	for i, call := range calls {
		out[i] = call
		if len(call.Args) > 0 {
			out[i].Args = append([]byte(nil), call.Args...)
		}
	}
	return out
}

func cloneToolResults(results []schema.ToolResult) []schema.ToolResult {
	if len(results) == 0 {
		return nil
	}
	out := make([]schema.ToolResult, len(results))
	for i, result := range results {
		out[i] = result
		if len(result.Result) > 0 {
			out[i].Result = append([]byte(nil), result.Result...)
		}
	}
	return out
}
