package mas

import (
	"context"
	"fmt"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

// Query is a minimal entry point: build Agent + Runner and run once.
func Query(ctx context.Context, model llm.ChatModel, input string, opts ...Option) (schema.Message, error) {
	result, err := QueryWithResult(ctx, model, input, opts...)
	if err != nil {
		return schema.Message{}, err
	}
	return result.Message, nil
}

// QueryStream is a minimal streaming entry point.
func QueryStream(ctx context.Context, model llm.ChatModel, input string, opts ...Option) (<-chan schema.StreamEvent, error) {
	common := applyOptions(opts...)
	return runStream(ctx, model, input, common)
}

// QueryWithResult returns a richer execution result.
func QueryWithResult(ctx context.Context, model llm.ChatModel, input string, opts ...Option) (runner.RunResult, error) {
	common := applyOptions(opts...)
	return runWithResult(ctx, model, input, common)
}

func runWithResult(ctx context.Context, model llm.ChatModel, input string, opts options) (runner.RunResult, error) {
	if model == nil {
		return runner.RunResult{}, fmt.Errorf("mas: model is nil")
	}

	agentID := opts.AgentID
	if agentID == "" {
		agentID = "assistant"
	}
	agentName := opts.AgentName
	if agentName == "" {
		agentName = "assistant"
	}

	ag := agent.New(
		agentID,
		agentName,
		agent.WithSystemPrompt(opts.SystemPrompt),
		agent.WithTools(opts.Tools...),
	)

	r := runner.New(runner.Config{
		Model:          model,
		Memory:         opts.Memory,
		ToolInvoker:    opts.ToolInvoker,
		Middlewares:    opts.Middlewares,
		Observer:       opts.Observer,
		Tracer:         opts.Tracer,
		ResponseFormat: opts.ResponseFormat,
		MaxTurns:       opts.MaxTurns,
		HistoryWindow:  opts.HistoryWindow,
	})

	return r.RunWithResult(ctx, ag, schema.Message{
		Role:    schema.RoleUser,
		Content: input,
	})
}

func runStream(ctx context.Context, model llm.ChatModel, input string, opts options) (<-chan schema.StreamEvent, error) {
	if model == nil {
		return nil, fmt.Errorf("mas: model is nil")
	}

	agentID := opts.AgentID
	if agentID == "" {
		agentID = "assistant"
	}
	agentName := opts.AgentName
	if agentName == "" {
		agentName = "assistant"
	}

	ag := agent.New(
		agentID,
		agentName,
		agent.WithSystemPrompt(opts.SystemPrompt),
		agent.WithTools(opts.Tools...),
	)

	r := runner.New(runner.Config{
		Model:          model,
		Memory:         opts.Memory,
		ToolInvoker:    opts.ToolInvoker,
		Middlewares:    opts.Middlewares,
		Observer:       opts.Observer,
		Tracer:         opts.Tracer,
		ResponseFormat: opts.ResponseFormat,
		MaxTurns:       opts.MaxTurns,
		HistoryWindow:  opts.HistoryWindow,
	})

	return r.RunStream(ctx, ag, schema.Message{
		Role:    schema.RoleUser,
		Content: input,
	})
}
