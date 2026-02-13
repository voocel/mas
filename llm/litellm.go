package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/voocel/agentcore"
	"github.com/voocel/litellm"
)

// LiteLLMAdapter adapts litellm to the llm.ChatModel interface.
type LiteLLMAdapter struct {
	*BaseModel
	client *litellm.Client
	model  string
}

// NewLiteLLMAdapter creates an adapter from a litellm Client.
func NewLiteLLMAdapter(model string, client *litellm.Client) *LiteLLMAdapter {
	modelInfo := ModelInfo{
		Name:     model,
		Provider: client.ProviderName(),
		Capabilities: []string{
			string(CapabilityChat),
			string(CapabilityCompletion),
			string(CapabilityStreaming),
			string(CapabilityToolCalling),
		},
	}

	// Enrich from registry if available
	if caps, ok := litellm.GetModelCapabilities(model); ok {
		modelInfo.MaxTokens = caps.MaxOutputTokens
		modelInfo.ContextSize = caps.MaxInputTokens
	}

	return &LiteLLMAdapter{
		BaseModel: NewBaseModel(modelInfo, DefaultGenerationConfig),
		client:    client,
		model:     model,
	}
}

// NewOpenAIModel creates an OpenAI adapter.
func NewOpenAIModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	cfg := litellm.ProviderConfig{APIKey: apiKey}
	if len(baseURL) > 0 {
		cfg.BaseURL = baseURL[0]
	}
	client, _ := litellm.NewWithProvider("openai", cfg)
	return NewLiteLLMAdapter(model, client)
}

// NewAnthropicModel creates an Anthropic adapter.
func NewAnthropicModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	cfg := litellm.ProviderConfig{APIKey: apiKey}
	if len(baseURL) > 0 {
		cfg.BaseURL = baseURL[0]
	}
	client, _ := litellm.NewWithProvider("anthropic", cfg)
	return NewLiteLLMAdapter(model, client)
}

// NewGeminiModel creates a Gemini adapter.
func NewGeminiModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	cfg := litellm.ProviderConfig{APIKey: apiKey}
	if len(baseURL) > 0 {
		cfg.BaseURL = baseURL[0]
	}
	client, _ := litellm.NewWithProvider("gemini", cfg)
	return NewLiteLLMAdapter(model, client)
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
			partial      = agentcore.Message{Role: agentcore.RoleAssistant}
			textStarted  bool
			thinkStarted bool
			finishReason string
			toolAcc      = litellm.NewToolCallAccumulator()
			toolStarted  = make(map[int]bool)
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
					partial.Content = append(partial.Content, agentcore.ThinkingBlock(""))
					idx := len(partial.Content) - 1
					eventChan <- StreamEvent{
						Type:         StreamEventThinkingStart,
						ContentIndex: idx,
						Message:      partial,
					}
				}
				idx := lastBlockIndex(partial.Content, agentcore.ContentThinking)
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
					partial.Content = append(partial.Content, agentcore.TextBlock(""))
					idx := len(partial.Content) - 1
					eventChan <- StreamEvent{
						Type:         StreamEventTextStart,
						ContentIndex: idx,
						Message:      partial,
					}
				}
				idx := lastBlockIndex(partial.Content, agentcore.ContentText)
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
				deltaIdx := chunk.ToolCallDelta.Index
				wasStarted := toolAcc.Started(deltaIdx)
				toolAcc.Apply(chunk.ToolCallDelta)

				if !wasStarted && !toolStarted[deltaIdx] {
					toolStarted[deltaIdx] = true
					eventChan <- StreamEvent{
						Type:    StreamEventToolCallStart,
						Message: partial,
					}
				}

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
			idx := lastBlockIndex(partial.Content, agentcore.ContentThinking)
			if idx >= 0 {
				eventChan <- StreamEvent{
					Type:         StreamEventThinkingEnd,
					ContentIndex: idx,
					Message:      partial,
				}
			}
		}
		if textStarted {
			idx := lastBlockIndex(partial.Content, agentcore.ContentText)
			if idx >= 0 {
				eventChan <- StreamEvent{
					Type:         StreamEventTextEnd,
					ContentIndex: idx,
					Message:      partial,
				}
			}
		}

		// Build final tool calls from accumulated deltas
		if calls := toolAcc.Build(); len(calls) > 0 {
			for _, tc := range calls {
				partial.Content = append(partial.Content, agentcore.ToolCallBlock(ToolCall{
					ID:   tc.ID,
					Name: tc.Function.Name,
					Args: []byte(tc.Function.Arguments),
				}))
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
func lastBlockIndex(blocks []agentcore.ContentBlock, ct agentcore.ContentType) int {
	for i := len(blocks) - 1; i >= 0; i-- {
		if blocks[i].Type == ct {
			return i
		}
	}
	return -1
}

// convertMessages converts agentcore.Message to litellm.Message.
func convertMessages(messages []Message) []litellm.Message {
	llmMessages := make([]litellm.Message, len(messages))
	for i, msg := range messages {
		llmMsg := litellm.Message{
			Role:    string(msg.Role),
			Content: msg.TextContent(),
		}

		if msg.Role == agentcore.RoleTool {
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

// convertResponse converts litellm.Response to agentcore.Message with content blocks.
func convertResponse(response *litellm.Response) Message {
	var content []agentcore.ContentBlock

	// Thinking/reasoning content
	if response.Reasoning != nil && response.Reasoning.Content != "" {
		content = append(content, agentcore.ThinkingBlock(response.Reasoning.Content))
	}

	// Text content
	if response.Content != "" {
		content = append(content, agentcore.TextBlock(response.Content))
	}

	// Tool calls
	for _, call := range response.ToolCalls {
		content = append(content, agentcore.ToolCallBlock(agentcore.ToolCall{
			ID:   call.ID,
			Name: call.Function.Name,
			Args: []byte(call.Function.Arguments),
		}))
	}

	// Map usage
	var usage *agentcore.Usage
	if response.Usage.TotalTokens > 0 {
		usage = &agentcore.Usage{
			Input:       response.Usage.PromptTokens,
			Output:      response.Usage.CompletionTokens,
			CacheRead:   response.Usage.CacheReadInputTokens,
			CacheWrite:  response.Usage.CacheCreationInputTokens,
			TotalTokens: response.Usage.TotalTokens,
		}
	}

	return Message{
		Role:       agentcore.RoleAssistant,
		Content:    content,
		StopReason: mapStopReason(response.FinishReason),
		Usage:      usage,
	}
}

// mapStopReason maps litellm canonical FinishReason to MAS StopReason.
func mapStopReason(reason string) agentcore.StopReason {
	switch reason {
	case litellm.FinishReasonStop, "":
		return agentcore.StopReasonStop
	case litellm.FinishReasonLength:
		return agentcore.StopReasonLength
	case litellm.FinishReasonToolCall:
		return agentcore.StopReasonToolUse
	case litellm.FinishReasonError:
		return agentcore.StopReasonError
	default:
		return agentcore.StopReason(reason)
	}
}

// applyThinkingConfig resolves CallOptions and sets ThinkingConfig on the request.
func applyThinkingConfig(req *litellm.Request, opts []CallOption) {
	callCfg := agentcore.ResolveCallConfig(opts)
	if callCfg.ThinkingLevel == "" || callCfg.ThinkingLevel == agentcore.ThinkingOff {
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
