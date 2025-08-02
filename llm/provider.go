package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/voocel/litellm"
)

// Provider represents an LLM provider interface
type Provider interface {
	// Chat sends a chat completion request
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Model returns the model name
	Model() string

	// Close closes the provider connection
	Close() error
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []Message              `json:"messages"`
	Model       string                 `json:"model,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Tools       []ToolDefinition       `json:"tools,omitempty"`
	ToolChoice  string                 `json:"tool_choice,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Message represents a chat message
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Choice represents a choice in the response
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
	Delta        Message `json:"delta,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ToolDefinition represents a tool definition for function calling
type ToolDefinition struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef represents a function definition
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ProviderConfig contains configuration for LLM providers
type ProviderConfig struct {
	APIKey      string        `json:"api_key"`
	BaseURL     string        `json:"base_url,omitempty"`
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Timeout     time.Duration `json:"timeout,omitempty"`
	Retries     int           `json:"retries,omitempty"`
}

// DefaultProviderConfig returns a default provider configuration
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   2000,
		Timeout:     30 * time.Second,
		Retries:     3,
	}
}

// NewProvider creates a new LLM provider based on the model name
func NewProvider(model, apiKey string) (Provider, error) {
	config := DefaultProviderConfig()
	config.Model = model
	config.APIKey = apiKey

	return NewProviderWithConfig(config)
}

// NewProviderWithConfig creates a new LLM provider with custom configuration
func NewProviderWithConfig(config ProviderConfig) (Provider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Create litellm client with provider-specific configuration
	var client *litellm.Client

	if isOpenAIModel(config.Model) {
		client = litellm.New(
			litellm.WithOpenAI(config.APIKey),
			litellm.WithDefaults(config.MaxTokens, config.Temperature),
		)
	} else if isAnthropicModel(config.Model) {
		client = litellm.New(
			litellm.WithAnthropic(config.APIKey),
			litellm.WithDefaults(config.MaxTokens, config.Temperature),
		)
	} else if isGeminiModel(config.Model) {
		client = litellm.New(
			litellm.WithGemini(config.APIKey),
			litellm.WithDefaults(config.MaxTokens, config.Temperature),
		)
	} else {
		// Default to OpenAI-compatible
		client = litellm.New(
			litellm.WithOpenAI(config.APIKey),
			litellm.WithDefaults(config.MaxTokens, config.Temperature),
		)
	}

	return &LiteLLMProvider{
		client: client,
		config: config,
	}, nil
}

// LiteLLMProvider implements Provider interface using the litellm library
type LiteLLMProvider struct {
	client *litellm.Client
	config ProviderConfig
}

// Chat implements the chat completion using litellm
func (p *LiteLLMProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Set defaults from provider config
	if req.Model == "" {
		req.Model = p.config.Model
	}
	if req.Temperature == 0 {
		req.Temperature = p.config.Temperature
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = p.config.MaxTokens
	}

	// Convert our request format to litellm format
	litellmReq := &litellm.Request{
		Model:    req.Model,
		Messages: convertMessagesToLiteLLM(req.Messages),
		Tools:    convertToolsToLiteLLM(req.Tools),
	}

	// Set optional parameters using pointer helpers
	if req.Temperature != 0 {
		litellmReq.Temperature = litellm.Float64Ptr(req.Temperature)
	}
	if req.MaxTokens != 0 {
		litellmReq.MaxTokens = litellm.IntPtr(req.MaxTokens)
	}

	// Make the request
	resp, err := p.client.Complete(ctx, litellmReq)
	if err != nil {
		return nil, fmt.Errorf("litellm chat completion failed: %w", err)
	}

	// Convert response back to our format
	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())

	var choices []Choice
	if resp.ToolCalls != nil && len(resp.ToolCalls) > 0 {
		// Response contains tool calls
		choices = []Choice{
			{
				Index: 0,
				Message: Message{
					Role:      "assistant",
					Content:   resp.Content,
					ToolCalls: convertToolCallsFromLiteLLM(resp.ToolCalls),
				},
				FinishReason: "tool_calls",
			},
		}
	} else {
		// Regular text response
		choices = []Choice{
			{
				Index: 0,
				Message: Message{
					Role:    "assistant",
					Content: resp.Content,
				},
				FinishReason: "stop",
			},
		}
	}

	return &ChatResponse{
		ID:      id,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: choices,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.PromptTokens + resp.Usage.CompletionTokens,
		},
	}, nil
}

// Model returns the model name
func (p *LiteLLMProvider) Model() string {
	return p.config.Model
}

// Close closes the provider connection
func (p *LiteLLMProvider) Close() error {
	// litellm client may have cleanup methods
	return nil
}

// Helper functions for converting between formats

// convertMessagesToLiteLLM converts our message format to litellm format
func convertMessagesToLiteLLM(messages []Message) []litellm.Message {
	result := make([]litellm.Message, len(messages))
	for i, msg := range messages {
		result[i] = litellm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// convertToolsToLiteLLM converts our tool format to litellm format
func convertToolsToLiteLLM(tools []ToolDefinition) []litellm.Tool {
	if len(tools) == 0 {
		return nil
	}

	result := make([]litellm.Tool, len(tools))
	for i, tool := range tools {
		result[i] = litellm.Tool{
			Type: tool.Type,
			Function: litellm.FunctionSchema{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	return result
}

// convertToolCallsFromLiteLLM converts litellm tool calls to our format
func convertToolCallsFromLiteLLM(toolCalls []litellm.ToolCall) []ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

// isOpenAIModel checks if the model is an OpenAI model
func isOpenAIModel(model string) bool {
	openaiModels := []string{
		"o3", "o4-mini", "gpt-4.1", "gpt-4.1-mini", "gpt-4o", "gpt-4o-mini",
	}

	for _, m := range openaiModels {
		if strings.HasPrefix(model, m) {
			return true
		}
	}
	return false
}

// isAnthropicModel checks if the model is an Anthropic model
func isAnthropicModel(model string) bool {
	anthropicModels := []string{
		"claude-3.7-sonnet", "claude-4-sonnet", "claude-4-opus",
	}

	for _, m := range anthropicModels {
		if strings.HasPrefix(model, m) {
			return true
		}
	}
	return false
}

// isGeminiModel checks if the model is a Gemini model
func isGeminiModel(model string) bool {
	geminiModels := []string{
		"gemini-2.5-pro", "gemini-2.5-flash",
	}

	for _, m := range geminiModels {
		if strings.HasPrefix(model, m) {
			return true
		}
	}
	return false
}

// Convenience functions for creating specific providers

// NewOpenAIProvider creates an OpenAI provider
func NewOpenAIProvider(config ProviderConfig) (Provider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	return NewProviderWithConfig(config)
}

// NewAnthropicProvider creates an Anthropic provider
func NewAnthropicProvider(config ProviderConfig) (Provider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	return NewProviderWithConfig(config)
}

// NewGeminiProvider creates a Gemini provider
func NewGeminiProvider(config ProviderConfig) (Provider, error) {
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com"
	}
	return NewProviderWithConfig(config)
}
