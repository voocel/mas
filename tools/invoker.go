package tools

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/voocel/mas/schema"
)

// Invoker executes tool calls.
type Invoker interface {
	Invoke(ctx context.Context, registry *Registry, calls []schema.ToolCall) ([]schema.ToolResult, error)
}

// SerialInvoker executes tools serially.
type SerialInvoker struct{}

// NewSerialInvoker creates a serial invoker.
func NewSerialInvoker() *SerialInvoker {
	return &SerialInvoker{}
}

func (i *SerialInvoker) Invoke(ctx context.Context, registry *Registry, calls []schema.ToolCall) ([]schema.ToolResult, error) {
	results := make([]schema.ToolResult, len(calls))
	var firstErr error
	for idx, call := range calls {
		result, err := executeToolCall(ctx, registry, call)
		results[idx] = result
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return results, firstErr
}

// ConcurrentInvoker executes tools concurrently with a limit.
type ConcurrentInvoker struct {
	MaxConcurrency int
}

// NewConcurrentInvoker creates a concurrent invoker.
func NewConcurrentInvoker(maxConcurrency int) *ConcurrentInvoker {
	if maxConcurrency <= 0 {
		maxConcurrency = 4
	}
	return &ConcurrentInvoker{MaxConcurrency: maxConcurrency}
}

func (i *ConcurrentInvoker) Invoke(ctx context.Context, registry *Registry, calls []schema.ToolCall) ([]schema.ToolResult, error) {
	if len(calls) == 0 {
		return []schema.ToolResult{}, nil
	}

	results := make([]schema.ToolResult, len(calls))
	errCh := make(chan error, len(calls))

	max := i.MaxConcurrency
	if max <= 0 || max > len(calls) {
		max = len(calls)
	}
	sem := make(chan struct{}, max)

	var wg sync.WaitGroup
	for idx, call := range calls {
		wg.Add(1)
		go func(i int, c schema.ToolCall) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[i] = schema.ToolResult{ID: c.ID, Error: ctx.Err().Error()}
				errCh <- ctx.Err()
				return
			}

			result, err := executeToolCall(ctx, registry, c)
			results[i] = result
			if err != nil {
				errCh <- err
			}
		}(idx, call)
	}

	wg.Wait()
	close(errCh)

	var firstErr error
	for err := range errCh {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return results, firstErr
}

func executeToolCall(ctx context.Context, registry *Registry, call schema.ToolCall) (schema.ToolResult, error) {
	if registry == nil {
		return schema.ToolResult{ID: call.ID, Error: "tool registry is nil"}, errors.New("tool registry is nil")
	}

	tool, exists := registry.Get(call.Name)
	if !exists {
		err := schema.NewToolError(call.Name, "execute", schema.ErrToolNotFound)
		return schema.ToolResult{ID: call.ID, Error: err.Error()}, err
	}

	if validator, ok := tool.(interface {
		ValidateInput(json.RawMessage) error
	}); ok {
		if err := validator.ValidateInput(call.Args); err != nil {
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

func getToolConfig(tool Tool) *ToolConfig {
	if baseTool, ok := tool.(*BaseTool); ok {
		return baseTool.Config()
	}
	type configGetter interface {
		Config() *ToolConfig
	}
	if getter, ok := tool.(configGetter); ok {
		if cfg := getter.Config(); cfg != nil {
			return cfg
		}
		return cloneToolConfig(DefaultToolConfig)
	}
	return cloneToolConfig(DefaultToolConfig)
}
