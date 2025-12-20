package llm

import (
	"context"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/voocel/litellm"
	"github.com/voocel/mas/schema"
)

// LiteLLMAdapter is a litellm adapter.
type LiteLLMAdapter struct {
	*BaseModel
	client *litellm.Client
	model  string
}

// NewLiteLLMAdapter creates an adapter.
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

// NewOpenAIModel creates an OpenAI adapter.
func NewOpenAIModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	var options []litellm.ClientOption
	if len(baseURL) > 0 {
		options = append(options, litellm.WithOpenAI(apiKey, baseURL[0]))
	} else {
		options = append(options, litellm.WithOpenAI(apiKey))
	}
	return NewLiteLLMAdapter(model, options...)
}

// NewAnthropicModel creates an Anthropic adapter.
func NewAnthropicModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	var options []litellm.ClientOption
	if len(baseURL) > 0 {
		options = append(options, litellm.WithAnthropic(apiKey, baseURL[0]))
	} else {
		options = append(options, litellm.WithAnthropic(apiKey))
	}
	return NewLiteLLMAdapter(model, options...)
}

// NewGeminiModel creates a Gemini adapter.
func NewGeminiModel(model, apiKey string, baseURL ...string) *LiteLLMAdapter {
	var options []litellm.ClientOption
	if len(baseURL) > 0 {
		options = append(options, litellm.WithGemini(apiKey, baseURL[0]))
	} else {
		options = append(options, litellm.WithGemini(apiKey))
	}
	return NewLiteLLMAdapter(model, options...)
}

// Generate produces a response.
func (l *LiteLLMAdapter) Generate(ctx context.Context, req *Request) (*Response, error) {
	// Merge config
	cfg := l.config
	if req != nil && req.Config != nil {
		cfg = req.Config
	}

	// Convert messages
	llmMessages, err := l.convertMessages(req.Messages)
	if err != nil {
		return nil, schema.NewModelError(l.info.Name, "convert_messages", err)
	}

	ltReq := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
		Stream:      false,
	}

	l.applyToolConfigFromRequest(req, ltReq)

	ltResp, err := l.client.Chat(ctx, ltReq)
	if err != nil {
		return nil, schema.NewModelError(l.info.Name, "chat", err)
	}

	msg, usage := l.convertResponse(ltResp)
	return &Response{
		Message:      msg,
		Usage:        usage,
		FinishReason: ltResp.FinishReason,
		ModelInfo:    l.info,
	}, nil
}

// GenerateStream produces a streaming response.
func (l *LiteLLMAdapter) GenerateStream(ctx context.Context, req *Request) (<-chan schema.StreamEvent, error) {
	cfg := l.config
	if req != nil && req.Config != nil {
		cfg = req.Config
	}

	llmMessages, err := l.convertMessages(req.Messages)
	if err != nil {
		return nil, schema.NewModelError(l.info.Name, "convert_messages", err)
	}

	request := &litellm.Request{
		Model:       l.model,
		Messages:    llmMessages,
		Temperature: &cfg.Temperature,
		MaxTokens:   &cfg.MaxTokens,
		Stream:      true,
	}

	l.applyToolConfigFromRequest(req, request)

	// Call the litellm streaming interface.
	stream, err := l.client.Stream(ctx, request)
	if err != nil {
		return nil, schema.NewModelError(l.info.Name, "stream", err)
	}

	eventChan := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(eventChan)
		defer stream.Close()

		eventChan <- schema.NewStreamEvent(schema.EventStart, nil)

		var fullContent string
		finishReason := ""
		toolCallBuilders := make(map[int]*toolCallBuilder)

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

			if chunk == nil {
				continue
			}

			if chunk.Content != "" {
				fullContent += chunk.Content

				// Send token event.
				eventChan <- schema.NewTokenEvent(fullContent, chunk.Content, "")
			}

			if chunk.ToolCallDelta != nil {
				applyToolCallDelta(toolCallBuilders, chunk.ToolCallDelta)
			}

			if chunk.FinishReason != "" {
				finishReason = chunk.FinishReason
			}
		}

		// Send end event.
		finalMessage := schema.Message{
			Role:      schema.RoleAssistant,
			Content:   fullContent,
			Timestamp: time.Now(),
		}

		if finishReason != "" {
			finalMessage.SetMetadata("finish_reason", finishReason)
		}

		if toolCalls := buildToolCalls(toolCallBuilders); len(toolCalls) > 0 {
			finalMessage.ToolCalls = toolCalls
		}
		eventChan <- schema.NewStreamEvent(schema.EventEnd, finalMessage)
	}()

	return eventChan, nil
}

// convertMessages converts message formats.
func (l *LiteLLMAdapter) convertMessages(messages []schema.Message) ([]litellm.Message, error) {
	llmMessages := make([]litellm.Message, len(messages))

	for i, msg := range messages {
		llmMsg := litellm.Message{
			Role:    string(msg.Role),
			Content: msg.Content,
		}

		if msg.Role == schema.RoleTool && msg.ID != "" {
			llmMsg.ToolCallID = msg.ID
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

	return llmMessages, nil
}

// convertResponse converts the response.
func (l *LiteLLMAdapter) convertResponse(response *litellm.Response) (schema.Message, TokenUsage) {
	message := schema.Message{
		Role:      schema.RoleAssistant,
		Content:   response.Content,
		Timestamp: time.Now(),
	}

	if len(response.ToolCalls) > 0 {
		message.ToolCalls = make([]schema.ToolCall, len(response.ToolCalls))
		for i, call := range response.ToolCalls {
			message.ToolCalls[i] = schema.ToolCall{
				ID:   call.ID,
				Name: call.Function.Name,
				Args: []byte(call.Function.Arguments),
			}
		}
	}

	usage := TokenUsage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}
	return message, usage
}

func (l *LiteLLMAdapter) applyToolConfigFromRequest(req *Request, request *litellm.Request) {
	if req == nil || request == nil {
		return
	}
	if len(req.Tools) > 0 {
		tools := make([]litellm.Tool, 0, len(req.Tools))
		for _, t := range req.Tools {
			if t.Name == "" {
				continue
			}
			tools = append(tools, litellm.Tool{
				Type: "function",
				Function: litellm.FunctionDef{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			})
		}
		request.Tools = tools
	}
	if req.ToolChoice != nil {
		switch req.ToolChoice.Type {
		case "auto":
			request.ToolChoice = "auto"
		case "none":
			request.ToolChoice = "none"
		case "required":
			request.ToolChoice = map[string]string{"type": "required"}
		case "function":
			if req.ToolChoice.Name != "" {
				request.ToolChoice = map[string]interface{}{
					"type":     "function",
					"function": map[string]string{"name": req.ToolChoice.Name},
				}
			}
		}
	}
	if req.ResponseFormat != nil {
		rf := &litellm.ResponseFormat{Type: req.ResponseFormat.Type}
		if js, ok := req.ResponseFormat.JSONSchema.(*litellm.JSONSchema); ok {
			rf.JSONSchema = js
		}
		request.ResponseFormat = rf
	}
	if req.Extra != nil {
		request.Extra = req.Extra
	}
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

func buildToolCalls(builders map[int]*toolCallBuilder) []schema.ToolCall {
	if len(builders) == 0 {
		return nil
	}

	indexes := make([]int, 0, len(builders))
	for idx := range builders {
		indexes = append(indexes, idx)
	}
	sort.Ints(indexes)

	toolCalls := make([]schema.ToolCall, 0, len(indexes))
	for _, idx := range indexes {
		builder := builders[idx]
		if builder == nil {
			continue
		}

		call := schema.ToolCall{
			ID:   builder.id,
			Name: builder.functionName,
			Args: []byte(builder.arguments.String()),
		}
		toolCalls = append(toolCalls, call)
	}

	return toolCalls
}

// getMaxTokens returns the model max tokens.
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
	// Context size usually equals max tokens.
	return getMaxTokens(model)
}

// extractProvider infers provider from model name.
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

// supportsToolCalling checks whether tool calling is supported.
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
