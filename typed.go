package mas

import (
	"context"
	"fmt"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/typed"
)

// TypedResult wraps the parsed result with execution metadata.
type TypedResult[T any] struct {
	Data        T
	Usage       llm.TokenUsage
	ToolCalls   []schema.ToolCall
	ToolResults []schema.ToolResult
}

// QueryTyped executes a query and returns a typed result.
// The response is automatically parsed into the specified type T.
func QueryTyped[T any](ctx context.Context, model llm.ChatModel, input string, opts ...Option) (T, error) {
	result, err := QueryTypedWithResult[T](ctx, model, input, opts...)
	if err != nil {
		var zero T
		return zero, err
	}
	return result.Data, nil
}

// QueryTypedWithResult executes a query and returns a typed result with metadata.
func QueryTypedWithResult[T any](ctx context.Context, model llm.ChatModel, input string, opts ...Option) (TypedResult[T], error) {
	var zero TypedResult[T]
	common := applyOptions(opts...)

	typeName := typed.CleanTypeName[T]()
	common.ResponseFormat = typed.ResponseFormatFromType[T](typeName)

	runResult, err := runWithResult(ctx, model, input, common)
	if err != nil {
		return zero, err
	}

	data, err := typed.ParseResponse[T](runResult.Message.Content)
	if err != nil {
		return zero, fmt.Errorf("failed to parse typed response: %w", err)
	}

	return TypedResult[T]{
		Data:        data,
		Usage:       runResult.Usage,
		ToolCalls:   runResult.ToolCalls,
		ToolResults: runResult.ToolResults,
	}, nil
}

// RunTyped executes an agent with typed output.
// Note: The runner must be configured with appropriate ResponseFormat for structured output.
// Consider using QueryTyped for automatic schema generation.
func RunTyped[T any](ctx context.Context, r *runner.Runner, ag *agent.Agent, input schema.Message) (T, error) {
	result, err := RunTypedWithResult[T](ctx, r, ag, input)
	if err != nil {
		var zero T
		return zero, err
	}
	return result.Data, nil
}

// RunTypedWithResult executes an agent with typed output and returns metadata.
// Note: The runner must be configured with appropriate ResponseFormat for structured output.
func RunTypedWithResult[T any](ctx context.Context, r *runner.Runner, ag *agent.Agent, input schema.Message) (TypedResult[T], error) {
	var zero TypedResult[T]

	runResult, err := r.RunWithResult(ctx, ag, input)
	if err != nil {
		return zero, err
	}

	data, err := typed.ParseResponse[T](runResult.Message.Content)
	if err != nil {
		return zero, fmt.Errorf("failed to parse typed response (ensure runner has ResponseFormat configured): %w", err)
	}

	return TypedResult[T]{
		Data:        data,
		Usage:       runResult.Usage,
		ToolCalls:   runResult.ToolCalls,
		ToolResults: runResult.ToolResults,
	}, nil
}
