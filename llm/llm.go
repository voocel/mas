package llm

import (
	"context"
)

type Message struct {
	Role    string                 `json:"role"`
	Content string                 `json:"content"`
	Name    string                 `json:"name,omitempty"`
	Images  []string               `json:"images,omitempty"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []Choice               `json:"choices"`
	Usage   Usage                  `json:"usage"`
	Extra   map[string]interface{} `json:"extra,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionRequest struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	Stop        []string               `json:"stop,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

type Provider interface {
	ID() string

	ChatCompletion(ctx context.Context, req ChatCompletionRequest) (*ChatCompletionResponse, error)

	GetModels(ctx context.Context) ([]string, error)

	Close() error
}

type Config struct {
	ProviderType string
	APIKey       string
	BaseURL      string
	DefaultModel string
	Timeout      int
	RetryCount   int
	Extra        map[string]interface{}
}

type Factory struct {
	providers map[string]func(Config) (Provider, error)
}

func NewFactory() *Factory {
	return &Factory{
		providers: make(map[string]func(Config) (Provider, error)),
	}
}

func (f *Factory) Register(
	providerType string,
	creator func(Config) (Provider, error),
) {
	f.providers[providerType] = creator
}

func (f *Factory) Create(config Config) (Provider, error) {
	creator, ok := f.providers[config.ProviderType]
	if !ok {
		return nil, ErrProviderNotSupported
	}
	return creator(config)
}

var (
	ErrProviderNotSupported = LLMError{Code: "provider_not_supported", Message: "LLM provider not supported"}
	ErrAPIKeyNotSet         = LLMError{Code: "api_key_not_set", Message: "API key not set"}
	ErrRequestFailed        = LLMError{Code: "request_failed", Message: "Request to LLM provider failed"}
	ErrResponseInvalid      = LLMError{Code: "response_invalid", Message: "Invalid response from LLM provider"}
)

type LLMError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e LLMError) Error() string {
	return e.Message
}

func (e LLMError) WithDetails(details string) LLMError {
	e.Details = details
	return e
}
