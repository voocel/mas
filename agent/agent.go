package agent

import (
	"context"
	"fmt"

	"github.com/voocel/mas/llm"
)

// agent is the default implementation of the Agent interface
type agent struct {
	config AgentConfig
}

// AgentConfig contains configuration for creating an agent
type AgentConfig struct {
	Name         string
	Model        string
	APIKey       string
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
	Tools        []Tool
	Memory       Memory
	State        map[string]interface{}
	Provider     llm.Provider
}

// Tool represents a tool interface (will be satisfied by mas.Tool)
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]any) (any, error)
	Schema() *ToolSchema
}

// ToolSchema represents tool schema (will be satisfied by mas.ToolSchema)
type ToolSchema struct {
	Type        string                     `json:"type"`
	Properties  map[string]*PropertySchema `json:"properties"`
	Required    []string                   `json:"required"`
	Description string                     `json:"description,omitempty"`
}

// PropertySchema defines a property in the tool schema
type PropertySchema struct {
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Enum        []string        `json:"enum,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
}

// Memory represents a memory interface (will be satisfied by mas.Memory)
type Memory interface {
	Add(ctx context.Context, role, content string) error
	GetHistory(ctx context.Context, limit int) ([]Message, error)
	Clear() error
	Count() int
}

// Message represents a message (will be satisfied by mas.Message)
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp interface{}            `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Agent interface represents the public API
type Agent interface {
	Chat(ctx context.Context, message string) (string, error)
	WithTools(tools ...Tool) Agent
	WithMemory(memory Memory) Agent
	WithSystemPrompt(prompt string) Agent
	WithTemperature(temp float64) Agent
	WithMaxTokens(tokens int) Agent
	SetState(key string, value interface{})
	GetState(key string) interface{}
	ClearState()
	Name() string
	Model() string
}

// DefaultAgentConfig returns a default configuration
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Name:        "assistant",
		Model:       "gpt-4.1-mini",
		Temperature: 0.7,
		MaxTokens:   2000,
		State:       make(map[string]interface{}),
	}
}

// New creates a new agent with minimal configuration
func New(model, apiKey string) Agent {
	config := DefaultAgentConfig()
	config.Model = model
	config.APIKey = apiKey

	// Create LLM provider
	provider, err := llm.NewProvider(model, apiKey)
	if err != nil {
		// For now, we'll create a fallback provider
		// In production, this should be handled more gracefully
		provider = nil
	}
	config.Provider = provider

	return NewWithConfig(config)
}

// NewWithConfig creates a new agent with full configuration
func NewWithConfig(config AgentConfig) Agent {
	if config.State == nil {
		config.State = make(map[string]interface{})
	}

	// Create provider if not provided
	if config.Provider == nil && config.APIKey != "" {
		provider, err := llm.NewProvider(config.Model, config.APIKey)
		if err == nil {
			config.Provider = provider
		}
	}

	return &agent{
		config: config,
	}
}

// Chat implements the core chat functionality
func (a *agent) Chat(ctx context.Context, message string) (string, error) {
	if a.config.Provider == nil {
		return "", fmt.Errorf("no LLM provider configured")
	}

	// Add user message to memory if available
	if a.config.Memory != nil {
		err := a.config.Memory.Add(ctx, "user", message)
		if err != nil {
			return "", fmt.Errorf("failed to add message to memory: %w", err)
		}
	}

	// Prepare messages for LLM
	messages, err := a.prepareMessages(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to prepare messages: %w", err)
	}

	// Prepare tools for function calling if available
	var tools []llm.ToolDefinition
	if len(a.config.Tools) > 0 {
		tools = a.convertToolsToLLMFormat()
	}

	// Create chat request
	req := llm.ChatRequest{
		Messages:    messages,
		Model:       a.config.Model,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
		Tools:       tools,
	}

	// Call LLM
	response, err := a.config.Provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	choice := response.Choices[0]

	// Handle tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, choice.Message, messages)
	}

	// Add assistant response to memory
	if a.config.Memory != nil {
		err := a.config.Memory.Add(ctx, "assistant", choice.Message.Content)
		if err != nil {
			return "", fmt.Errorf("failed to add response to memory: %w", err)
		}
	}

	return choice.Message.Content, nil
}

// Fluent configuration methods
func (a *agent) WithTools(tools ...Tool) Agent {
	newConfig := a.config
	newConfig.Tools = append(newConfig.Tools, tools...)
	return &agent{config: newConfig}
}

func (a *agent) WithMemory(memory Memory) Agent {
	newConfig := a.config
	newConfig.Memory = memory
	return &agent{config: newConfig}
}

func (a *agent) WithSystemPrompt(prompt string) Agent {
	newConfig := a.config
	newConfig.SystemPrompt = prompt
	return &agent{config: newConfig}
}

func (a *agent) WithTemperature(temp float64) Agent {
	newConfig := a.config
	newConfig.Temperature = temp
	return &agent{config: newConfig}
}

func (a *agent) WithMaxTokens(tokens int) Agent {
	newConfig := a.config
	newConfig.MaxTokens = tokens
	return &agent{config: newConfig}
}

// State management
func (a *agent) SetState(key string, value interface{}) {
	a.config.State[key] = value
}

func (a *agent) GetState(key string) interface{} {
	return a.config.State[key]
}

func (a *agent) ClearState() {
	a.config.State = make(map[string]interface{})
}

func (a *agent) Name() string {
	return a.config.Name
}

func (a *agent) Model() string {
	return a.config.Model
}
