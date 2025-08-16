package llm

import (
	"fmt"
	"strings"
	"time"
)

// DefaultProviderConfig returns a default provider configuration
func DefaultProviderConfig() ProviderConfig {
	return ProviderConfig{
		Model:       "gpt-4.1-mini",
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

	// For now, we default to LiteLLM provider
	// In the future, we can add routing logic for different providers
	return NewLiteLLMProvider(config)
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
