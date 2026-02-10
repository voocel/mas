package llm

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/voocel/litellm"
	"github.com/voocel/litellm/providers"
)

// LiteLLMAdapter adapts litellm to the llm.ChatModel interface.
type LiteLLMAdapter struct {
	*BaseModel
	client *litellm.Client
	model  string
}

// NewLiteLLMAdapter creates an adapter from a provider.
func NewLiteLLMAdapter(model string, provider providers.Provider, options ...litellm.ClientOption) *LiteLLMAdapter {
	client, _ := litellm.New(provider, options...)

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

	if supportsToolCalling(model) {
		modelInfo.Capabilities = append(modelInfo.Capabilities, string(CapabilityToolCalling))
	}

	return &LiteLLMAdapter{
		BaseModel: NewBaseModel(modelInfo, DefaultGenerationConfig),
		client:    client,
		model:     model,
	}
}

// NewOpenAIModel creates an OpenAI adapter.
func NewOpenAIModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	cfg := providers.ProviderConfig{APIKey: apiKey}
	if len(baseURL) > 0 {
		cfg.BaseURL = baseURL[0]
	}
	return NewLiteLLMAdapter(model, providers.NewOpenAI(cfg))
}

// NewAnthropicModel creates an Anthropic adapter.
func NewAnthropicModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	cfg := providers.ProviderConfig{APIKey: apiKey}
	if len(baseURL) > 0 {
		cfg.BaseURL = baseURL[0]
	}
	return NewLiteLLMAdapter(model, providers.NewAnthropic(cfg))
}

// NewGeminiModel creates a Gemini adapter.
func NewGeminiModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	cfg := providers.ProviderConfig{APIKey: apiKey}
	if len(baseURL) > 0 {
		cfg.BaseURL = baseURL[0]
	}
	return NewLiteLLMAdapter(model, providers.NewGemini(cfg))
}

// Generate produces a synchronous response.
func (l *LiteLLMAdapter) Generate(ctx context.Context, messages []Message, tools []ToolSpec) (*LLMResponse, error) {
	cfg := l.GetConfig()

	llmMessages := convertMessages(messages)

	ltReq := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
	}

	applyToolConfig(ltReq, tools)

	ltResp, err := l.client.Chat(ctx, ltReq)
	if err != nil {
		return nil, fmt.Errorf("llm: chat failed: %w", err)
	}

	msg := convertResponse(ltResp)
	return &LLMResponse{Message: msg}, nil
}

// GenerateStream produces a streaming response.
func (l *LiteLLMAdapter) GenerateStream(ctx context.Context, messages []Message, tools []ToolSpec) (<-chan StreamEvent, error) {
	cfg := l.GetConfig()

	llmMessages := convertMessages(messages)

	request := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
	}

	applyToolConfig(request, tools)

	stream, err := l.client.Stream(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("llm: stream failed: %w", err)
	}

	eventChan := make(chan StreamEvent, 100)

	go func() {
		defer close(eventChan)
		defer stream.Close()

		var fullContent string
		toolCallBuilders := make(map[int]*toolCallBuilder)

		for {
			chunk, err := stream.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				eventChan <- StreamEvent{Type: StreamEventError, Err: err}
				return
			}

			if chunk == nil {
				continue
			}

			if chunk.Content != "" {
				fullContent += chunk.Content
				eventChan <- StreamEvent{Type: StreamEventToken, Delta: chunk.Content}
			}

			if chunk.ToolCallDelta != nil {
				applyToolCallDelta(toolCallBuilders, chunk.ToolCallDelta)
			}
		}

		finalMessage := Message{
			Role:    RoleAssistant,
			Content: fullContent,
		}

		if toolCalls := buildToolCalls(toolCallBuilders); len(toolCalls) > 0 {
			finalMessage.ToolCalls = toolCalls
		}

		eventChan <- StreamEvent{Type: StreamEventDone, Message: finalMessage}
	}()

	return eventChan, nil
}

// convertMessages converts llm.Message to litellm.Message.
func convertMessages(messages []Message) []litellm.Message {
	llmMessages := make([]litellm.Message, len(messages))
	for i, msg := range messages {
		llmMsg := litellm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		if msg.Role == RoleTool {
			if id, ok := msg.Metadata["tool_call_id"].(string); ok {
				llmMsg.ToolCallID = id
			}
		}

		if len(msg.ToolCalls) > 0 {
			llmMsg.ToolCalls = make([]litellm.ToolCall, len(msg.ToolCalls))
			for idx, call := range msg.ToolCalls {
				llmMsg.ToolCalls[idx] = litellm.ToolCall{
					ID:   call.ID,
					Type: "function",
					Function: litellm.FunctionCall{
						Name:      call.Name,
						Arguments: string(call.Args),
					},
				}
			}
		}

		llmMessages[i] = llmMsg
	}
	return llmMessages
}

// convertResponse converts litellm.Response to llm.Message.
func convertResponse(response *litellm.Response) Message {
	message := Message{
		Role:    RoleAssistant,
		Content: response.Content,
	}

	if len(response.ToolCalls) > 0 {
		message.ToolCalls = make([]ToolCall, len(response.ToolCalls))
		for i, call := range response.ToolCalls {
			message.ToolCalls[i] = ToolCall{
				ID:   call.ID,
				Name: call.Function.Name,
				Args: []byte(call.Function.Arguments),
			}
		}
	}

	return message
}

func applyToolConfig(request *litellm.Request, tools []ToolSpec) {
	if len(tools) == 0 {
		return
	}
	ltTools := make([]litellm.Tool, 0, len(tools))
	for _, t := range tools {
		if t.Name == "" {
			continue
		}
		ltTools = append(ltTools, litellm.Tool{
			Type: "function",
			Function: litellm.FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	request.Tools = ltTools
	request.ToolChoice = "auto"
}

type toolCallBuilder struct {
	id           string
	callType     string
	functionName string
	arguments    strings.Builder
}

func applyToolCallDelta(builders map[int]*toolCallBuilder, delta *litellm.ToolCallDelta) {
	if builders == nil || delta == nil {
		return
	}

	builder, exists := builders[delta.Index]
	if !exists {
		builder = &toolCallBuilder{}
		builders[delta.Index] = builder
	}

	if delta.ID != "" {
		builder.id = delta.ID
	}
	if delta.Type != "" {
		builder.callType = delta.Type
	}
	if delta.FunctionName != "" {
		builder.functionName = delta.FunctionName
	}
	if delta.ArgumentsDelta != "" {
		builder.arguments.WriteString(delta.ArgumentsDelta)
	}
}

func buildToolCalls(builders map[int]*toolCallBuilder) []ToolCall {
	if len(builders) == 0 {
		return nil
	}

	indexes := make([]int, 0, len(builders))
	for idx := range builders {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)

	toolCalls := make([]ToolCall, 0, len(indexes))
	for _, idx := range indexes {
		builder := builders[idx]
		if builder == nil {
			continue
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:   builder.id,
			Name: builder.functionName,
			Args: []byte(builder.arguments.String()),
		})
	}
	return toolCalls
}

func getMaxTokens(model string) int {
	switch model {
	case "gpt-4.1", "gpt-4.1-mini":
		return 32000
	case "gpt-5", "gpt-5-mini":
		return 128000
	case "claude-3.7-sonnet", "claude-4-sonnet":
		return 128000
	case "gemini-2.5-pro", "gemini-2.5-flash":
		return 1048576
	default:
		return 4096
	}
}

func getContextSize(model string) int { return getMaxTokens(model) }

func extractProvider(model string) string {
	switch {
	case strings.HasPrefix(model, "gpt-"):
		return "openai"
	case strings.HasPrefix(model, "claude-"):
		return "anthropic"
	case strings.HasPrefix(model, "gemini-"):
		return "google"
	default:
		return "unknown"
	}
}

func supportsToolCalling(model string) bool {
	supported := []string{
		"gpt-5", "gpt-4.1", "gpt-4.1-mini", "gpt-4.1-nano", "gpt-4o", "gpt-4o-mini",
		"claude-4-sonnet", "claude-3.7-sonnet", "claude-3.5-haiku",
		"gemini-2.5-pro", "gemini-2.5-flash",
	}
	for _, s := range supported {
		if model == s {
			return true
		}
	}
	return false
}
