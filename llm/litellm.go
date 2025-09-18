package llm

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/voocel/litellm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// LiteLLMAdapter is an adapter for litellm.
type LiteLLMAdapter struct {
	*BaseModel
	client *litellm.Client
	model  string
}

// NewLiteLLMAdapter creates a new litellm adapter.
func NewLiteLLMAdapter(model string, options ...litellm.ClientOption) *LiteLLMAdapter {
	client := litellm.New(options...)

	modelInfo := ModelInfo{
		Name:        model,
		Provider:    extractProvider(model),
		Version:     "1.0",
		MaxTokens:   getMaxTokens(model),
		ContextSize: getContextSize(model),
		Capabilities: []string{
			string(CapabilityChat),
			string(CapabilityCompletion),
			string(CapabilityStreaming),
		},
	}

	// Check if tool calling is supported.
	if supportsToolCalling(model) {
		modelInfo.Capabilities = append(modelInfo.Capabilities, string(CapabilityToolCalling))
	}

	baseModel := NewBaseModel(modelInfo, DefaultGenerationConfig)

	return &LiteLLMAdapter{
		BaseModel: baseModel,
		client:    client,
		model:     model,
	}
}

// NewOpenAIModel creates an OpenAI model adapter.
func NewOpenAIModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	var options []litellm.ClientOption
	if len(baseURL) > 0 {
		options = append(options, litellm.WithOpenAI(apiKey, baseURL[0]))
	} else {
		options = append(options, litellm.WithOpenAI(apiKey))
	}
	return NewLiteLLMAdapter(model, options...)
}

// NewAnthropicModel creates an Anthropic model adapter.
func NewAnthropicModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	var options []litellm.ClientOption
	if len(baseURL) > 0 {
		options = append(options, litellm.WithAnthropic(apiKey, baseURL[0]))
	} else {
		options = append(options, litellm.WithAnthropic(apiKey))
	}
	return NewLiteLLMAdapter(model, options...)
}

// NewGeminiModel creates a Gemini model adapter.
func NewGeminiModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	var options []litellm.ClientOption
	if len(baseURL) > 0 {
		options = append(options, litellm.WithGemini(apiKey, baseURL[0]))
	} else {
		options = append(options, litellm.WithGemini(apiKey))
	}
	return NewLiteLLMAdapter(model, options...)
}

// Generate generates a response.
func (l *LiteLLMAdapter) Generate(ctx runtime.Context, messages []schema.Message) (schema.Message, error) {
	// Convert message format.
	llmMessages, err := l.convertMessages(messages)
	if err != nil {
		return schema.Message{}, schema.NewModelError(l.info.Name, "convert_messages", err)
	}

	// Build request.
	request := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &l.config.Temperature,
		MaxTokens:   &l.config.MaxTokens,
		Stream:      false,
	}

	// Call litellm.
	response, err := l.client.Chat(context.Background(), request)
	if err != nil {
		return schema.Message{}, schema.NewModelError(l.info.Name, "chat", err)
	}

	return l.convertResponse(response), nil
}

// GenerateStream generates a streaming response.
func (l *LiteLLMAdapter) GenerateStream(ctx runtime.Context, messages []schema.Message) (<-chan schema.StreamEvent, error) {
	llmMessages, err := l.convertMessages(messages)
	if err != nil {
		return nil, schema.NewModelError(l.info.Name, "convert_messages", err)
	}

	request := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &l.config.Temperature,
		MaxTokens:   &l.config.MaxTokens,
		Stream:      true,
	}

	// Call the litellm streaming interface.
	stream, err := l.client.Stream(context.Background(), request)
	if err != nil {
		return nil, schema.NewModelError(l.info.Name, "stream", err)
	}

	eventChan := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(eventChan)
		defer stream.Close()

		eventChan <- schema.NewStreamEvent(schema.EventStart, nil)

		var fullContent string

		// Process the streaming response.
		for {
			chunk, err := stream.Next()
			if err != nil {
				if err == io.EOF {
					break
				}

				eventChan <- schema.NewErrorEvent(err, "stream_next")
				return
			}

			if chunk.Content != "" {
				fullContent += chunk.Content

				// Send token event.
				eventChan <- schema.NewTokenEvent(fullContent, chunk.Content, "")
			}
		}

		// Send end event.
		finalMessage := schema.Message{
			Role:      schema.RoleAssistant,
			Content:   fullContent,
			Timestamp: time.Now(),
		}
		eventChan <- schema.NewStreamEvent(schema.EventEnd, finalMessage)
	}()

	return eventChan, nil
}

// convertMessages converts the message format.
func (l *LiteLLMAdapter) convertMessages(messages []schema.Message) ([]litellm.Message, error) {
	llmMessages := make([]litellm.Message, len(messages))

	for i, msg := range messages {
		llmMsg := litellm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		// TODO: Add support for tool calls.

		llmMessages[i] = llmMsg
	}

	return llmMessages, nil
}

// convertResponse converts the response.
func (l *LiteLLMAdapter) convertResponse(response *litellm.Response) schema.Message {
	message := schema.Message{
		Role:      schema.RoleAssistant,
		Content:   response.Content,
		Timestamp: time.Now(),
	}

	// TODO: Add support for tool calls.

	return message
}

// hasToolCalls checks if the messages contain tool calls.
func (l *LiteLLMAdapter) hasToolCalls(messages []schema.Message) bool {
	for _, msg := range messages {
		if len(msg.ToolCalls) > 0 {
			return true
		}
	}
	return false
}

// Get the maximum number of tokens for the model.
func getMaxTokens(model string) int {
	// Return the corresponding maximum number of tokens based on the model name.
	switch model {
	case "gpt-4.1", "gpt-4.1-mini":
		return 32000
	case "gpt-5", "gpt-5-mini":
		return 128000
	case "claude-3.7-sonnet":
		return 128000
	case "claude-4-sonnet":
		return 128000
	case "claude-3.5-haiku":
		return 4096
	case "gemini-2.5-pro":
		return 1048576
	case "gemini-2.5-flash":
		return 1048576
	default:
		return 4096 // Default value
	}
}

func getContextSize(model string) int {
	// Usually the context size is equal to the maximum number of tokens.
	return getMaxTokens(model)
}

// extractProvider extracts the provider from the model name.
func extractProvider(model string) string {
	if strings.HasPrefix(model, "gpt-") {
		return "openai"
	}
	if strings.HasPrefix(model, "claude-") {
		return "anthropic"
	}
	if strings.HasPrefix(model, "gemini-") {
		return "google"
	}
	if strings.HasPrefix(model, "llama-") {
		return "meta"
	}
	return "unknown"
}

// Check if the model supports tool calling.
func supportsToolCalling(model string) bool {
	supportedModels := []string{
		"gpt-5", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "gpt-4o", "gpt-4o-mini",
		"claude-4-sonnet", "claude-3.7-sonnet", "claude-3.5-haiku",
		"gemini-2.5-pro", "gemini-2.5-flash",
	}

	for _, supported := range supportedModels {
		if model == supported {
			return true
		}
	}
	return false
}
