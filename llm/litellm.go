package llm

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/voocel/litellm"
	"github.com/voocel/litellm/providers"
	"github.com/voocel/mas"
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
func (l *LiteLLMAdapter) Generate(ctx context.Context, messages []Message, tools []ToolSpec, opts ...CallOption) (*LLMResponse, error) {
	cfg := l.GetConfig()

	llmMessages := convertMessages(messages)

	ltReq := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
	}

	applyThinkingConfig(ltReq, opts)
	applyToolConfig(ltReq, tools)

	ltResp, err := l.client.Chat(ctx, ltReq)
	if err != nil {
		return nil, fmt.Errorf("llm: chat failed: %w", err)
	}

	msg := convertResponse(ltResp)
	return &LLMResponse{Message: msg}, nil
}

// GenerateStream produces a streaming response with fine-grained events.
func (l *LiteLLMAdapter) GenerateStream(ctx context.Context, messages []Message, tools []ToolSpec, opts ...CallOption) (<-chan StreamEvent, error) {
	cfg := l.GetConfig()

	llmMessages := convertMessages(messages)

	request := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
	}

	applyThinkingConfig(request, opts)
	applyToolConfig(request, tools)

	stream, err := l.client.Stream(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("llm: stream failed: %w", err)
	}

	eventChan := make(chan StreamEvent, 100)

	go func() {
		defer close(eventChan)
		defer stream.Close()

		var (
			partial        = mas.Message{Role: mas.RoleAssistant}
			textStarted    bool
			thinkStarted   bool
			finishReason   string
			toolBuilders   = make(map[int]*toolCallBuilder)
			toolStarted    = make(map[int]bool)
		)

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

			// Reasoning/thinking chunks
			if chunk.Reasoning != nil && chunk.Reasoning.Content != "" {
				if !thinkStarted {
					thinkStarted = true
					partial.Content = append(partial.Content, mas.ThinkingBlock(""))
					idx := len(partial.Content) - 1
					eventChan <- StreamEvent{
						Type:         StreamEventThinkingStart,
						ContentIndex: idx,
						Message:      partial,
					}
				}
				idx := lastBlockIndex(partial.Content, mas.ContentThinking)
				partial.Content[idx].Thinking += chunk.Reasoning.Content
				eventChan <- StreamEvent{
					Type:         StreamEventThinkingDelta,
					ContentIndex: idx,
					Delta:        chunk.Reasoning.Content,
					Message:      partial,
				}
			}

			// Text content chunks
			if chunk.Content != "" {
				if !textStarted {
					textStarted = true
					partial.Content = append(partial.Content, mas.TextBlock(""))
					idx := len(partial.Content) - 1
					eventChan <- StreamEvent{
						Type:         StreamEventTextStart,
						ContentIndex: idx,
						Message:      partial,
					}
				}
				idx := lastBlockIndex(partial.Content, mas.ContentText)
				partial.Content[idx].Text += chunk.Content
				eventChan <- StreamEvent{
					Type:         StreamEventTextDelta,
					ContentIndex: idx,
					Delta:        chunk.Content,
					Message:      partial,
				}
			}

			// Tool call deltas
			if chunk.ToolCallDelta != nil {
				applyToolCallDelta(toolBuilders, chunk.ToolCallDelta)

				// Emit toolcall_start for new tool calls
				deltaIdx := chunk.ToolCallDelta.Index
				if !toolStarted[deltaIdx] && toolBuilders[deltaIdx] != nil {
					toolStarted[deltaIdx] = true
					eventChan <- StreamEvent{
						Type:    StreamEventToolCallStart,
						Message: partial,
					}
				}

				// Emit toolcall_delta for argument increments
				if chunk.ToolCallDelta.ArgumentsDelta != "" {
					eventChan <- StreamEvent{
						Type:    StreamEventToolCallDelta,
						Delta:   chunk.ToolCallDelta.ArgumentsDelta,
						Message: partial,
					}
				}
			}

			if chunk.FinishReason != "" {
				finishReason = chunk.FinishReason
			}
		}

		// Emit end events for open blocks
		if thinkStarted {
			idx := lastBlockIndex(partial.Content, mas.ContentThinking)
			if idx >= 0 {
				eventChan <- StreamEvent{
					Type:         StreamEventThinkingEnd,
					ContentIndex: idx,
					Message:      partial,
				}
			}
		}
		if textStarted {
			idx := lastBlockIndex(partial.Content, mas.ContentText)
			if idx >= 0 {
				eventChan <- StreamEvent{
					Type:         StreamEventTextEnd,
					ContentIndex: idx,
					Message:      partial,
				}
			}
		}

		// Build final tool calls from accumulated deltas
		if toolCalls := buildToolCalls(toolBuilders); len(toolCalls) > 0 {
			for _, tc := range toolCalls {
				partial.Content = append(partial.Content, mas.ToolCallBlock(tc))
				idx := len(partial.Content) - 1
				eventChan <- StreamEvent{
					Type:         StreamEventToolCallEnd,
					ContentIndex: idx,
					Message:      partial,
				}
			}
		}

		partial.StopReason = mapStopReason(finishReason)
		eventChan <- StreamEvent{Type: StreamEventDone, Message: partial, StopReason: partial.StopReason}
	}()

	return eventChan, nil
}

// lastBlockIndex returns the index of the last ContentBlock of the given type.
func lastBlockIndex(blocks []mas.ContentBlock, ct mas.ContentType) int {
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].Type == ct {
			return i
		}
	}
	return -1
}

// convertMessages converts mas.Message to litellm.Message.
func convertMessages(messages []Message) []litellm.Message {
	llmMessages := make([]litellm.Message, len(messages))
	for i, msg := range messages {
		llmMsg := litellm.Message{
			Role:    string(msg.Role),
			Content: msg.TextContent(),
		}

		if msg.Role == mas.RoleTool {
			if id, ok := msg.Metadata["tool_call_id"].(string); ok {
				llmMsg.ToolCallID = id
			}
		}

		toolCalls := msg.ToolCalls()
		if len(toolCalls) > 0 {
			llmMsg.ToolCalls = make([]litellm.ToolCall, len(toolCalls))
			for idx, call := range toolCalls {
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

// convertResponse converts litellm.Response to mas.Message with content blocks.
func convertResponse(response *litellm.Response) Message {
	var content []mas.ContentBlock

	// Thinking/reasoning content
	if response.Reasoning != nil && response.Reasoning.Content != "" {
		content = append(content, mas.ThinkingBlock(response.Reasoning.Content))
	}

	// Text content
	if response.Content != "" {
		content = append(content, mas.TextBlock(response.Content))
	}

	// Tool calls
	for _, call := range response.ToolCalls {
		content = append(content, mas.ToolCallBlock(mas.ToolCall{
			ID:   call.ID,
			Name: call.Function.Name,
			Args: []byte(call.Function.Arguments),
		}))
	}

	// Map usage
	var usage *mas.Usage
	if response.Usage.TotalTokens > 0 {
		usage = &mas.Usage{
			Input:       response.Usage.PromptTokens,
			Output:      response.Usage.CompletionTokens,
			CacheRead:   response.Usage.CacheReadInputTokens,
			CacheWrite:  response.Usage.CacheCreationInputTokens,
			TotalTokens: response.Usage.TotalTokens,
		}
	}

	return Message{
		Role:       mas.RoleAssistant,
		Content:    content,
		StopReason: mapStopReason(response.FinishReason),
		Usage:      usage,
	}
}

// mapStopReason converts provider finish reasons to StopReason.
func mapStopReason(reason string) mas.StopReason {
	switch reason {
	case "stop", "end_turn":
		return mas.StopReasonStop
	case "length", "max_tokens":
		return mas.StopReasonLength
	case "tool_calls", "tool_use":
		return mas.StopReasonToolUse
	default:
		if reason != "" {
			return mas.StopReason(reason)
		}
		return mas.StopReasonStop
	}
}

// applyThinkingConfig resolves CallOptions and sets ThinkingConfig on the request.
func applyThinkingConfig(req *litellm.Request, opts []CallOption) {
	callCfg := mas.ResolveCallConfig(opts)
	if callCfg.ThinkingLevel == "" || callCfg.ThinkingLevel == mas.ThinkingOff {
		return
	}
	req.Thinking = litellm.NewThinkingWithLevel(string(callCfg.ThinkingLevel))
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
