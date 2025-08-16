package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/voocel/litellm"
)

// LiteLLMProvider implements Provider interface using the litellm library
type LiteLLMProvider struct {
	client *litellm.Client
	config ProviderConfig
}

// NewLiteLLMProvider creates a new LiteLLM provider
func NewLiteLLMProvider(config ProviderConfig) (Provider, error) {
	// Create litellm client with provider-specific configuration
	var client *litellm.Client

	if isOpenAIModel(config.Model) {
		if config.BaseURL != "" {
			client = litellm.New(
				litellm.WithOpenAI(config.APIKey, config.BaseURL),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		} else {
			client = litellm.New(
				litellm.WithOpenAI(config.APIKey),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		}
	} else if isAnthropicModel(config.Model) {
		if config.BaseURL != "" {
			client = litellm.New(
				litellm.WithAnthropic(config.APIKey, config.BaseURL),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		} else {
			client = litellm.New(
				litellm.WithAnthropic(config.APIKey),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		}
	} else if isGeminiModel(config.Model) {
		if config.BaseURL != "" {
			client = litellm.New(
				litellm.WithGemini(config.APIKey, config.BaseURL),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		} else {
			client = litellm.New(
				litellm.WithGemini(config.APIKey),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		}
	} else {
		// Default to OpenAI-compatible
		if config.BaseURL != "" {
			client = litellm.New(
				litellm.WithOpenAI(config.APIKey, config.BaseURL),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		} else {
			client = litellm.New(
				litellm.WithOpenAI(config.APIKey),
				litellm.WithDefaults(config.MaxTokens, config.Temperature),
			)
		}
	}

	return &LiteLLMProvider{
		client: client,
		config: config,
	}, nil
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