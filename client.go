package mas

import (
	"context"
	"fmt"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

// Client provides a session-style interface.
type Client struct {
	agent  *agent.Agent
	runner *runner.Runner
}

// NewClient creates a Client using options.
func NewClient(model llm.ChatModel, opts ...Option) (*Client, error) {
	common := applyOptions(opts...)
	if model == nil {
		return nil, fmt.Errorf("mas: model is nil")
	}

	agentID := common.AgentID
	if agentID == "" {
		agentID = "assistant"
	}
	agentName := common.AgentName
	if agentName == "" {
		agentName = "assistant"
	}

	ag := agent.New(
		agentID,
		agentName,
		agent.WithSystemPrompt(common.SystemPrompt),
		agent.WithTools(common.Tools...),
	)

	r := runner.New(runner.Config{
		Model:          model,
		Memory:         common.Memory,
		ToolInvoker:    common.ToolInvoker,
		Middlewares:    common.Middlewares,
		Observer:       common.Observer,
		Tracer:         common.Tracer,
		ResponseFormat: common.ResponseFormat,
		MaxTurns:       common.MaxTurns,
		HistoryWindow:  common.HistoryWindow,
	})

	return &Client{agent: ag, runner: r}, nil
}

// Send sends a single message and returns the response.
func (c *Client) Send(ctx context.Context, input string) (schema.Message, error) {
	if c == nil || c.runner == nil || c.agent == nil {
		return schema.Message{}, fmt.Errorf("mas: client not initialized")
	}
	return c.runner.Run(ctx, c.agent, schema.Message{
		Role:    schema.RoleUser,
		Content: input,
	})
}

// SendWithResult returns the full execution result.
func (c *Client) SendWithResult(ctx context.Context, input string) (runner.RunResult, error) {
	if c == nil || c.runner == nil || c.agent == nil {
		return runner.RunResult{}, fmt.Errorf("mas: client not initialized")
	}
	return c.runner.RunWithResult(ctx, c.agent, schema.Message{
		Role:    schema.RoleUser,
		Content: input,
	})
}

// SendStream sends a message and streams events.
func (c *Client) SendStream(ctx context.Context, input string) (<-chan schema.StreamEvent, error) {
	if c == nil || c.runner == nil || c.agent == nil {
		return nil, fmt.Errorf("mas: client not initialized")
	}
	return c.runner.RunStream(ctx, c.agent, schema.Message{
		Role:    schema.RoleUser,
		Content: input,
	})
}
