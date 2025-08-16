package mas

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/voocel/mas/llm"
)

type agent struct {
	name         string
	model        string
	systemPrompt string
	temperature  float64
	maxTokens    int
	tools        []Tool
	memory       Memory
	state        map[string]interface{}
	provider     llm.Provider
	mu           sync.RWMutex
}

func NewAgent(model, apiKey string) Agent {
	provider, err := llm.NewProvider(model, apiKey)
	if err != nil {
		provider = nil
	}

	return &agent{
		name:        "assistant",
		model:       model,
		temperature: 0.7,
		maxTokens:   2000,
		tools:       make([]Tool, 0),
		state:       make(map[string]interface{}),
		provider:    provider,
	}
}

func NewAgentWithConfig(config AgentConfig) Agent {
	var provider llm.Provider
	if config.Provider != nil {
		provider = nil
	}
	if provider == nil && config.APIKey != "" {
		providerConfig := llm.ProviderConfig{
			Model:       config.Model,
			APIKey:      config.APIKey,
			BaseURL:     config.BaseURL,
			Temperature: config.Temperature,
			MaxTokens:   config.MaxTokens,
		}
		provider, _ = llm.NewProviderWithConfig(providerConfig)
	}

	if config.State == nil {
		config.State = make(map[string]interface{})
	}

	return &agent{
		name:         config.Name,
		model:        config.Model,
		systemPrompt: config.SystemPrompt,
		temperature:  config.Temperature,
		maxTokens:    config.MaxTokens,
		tools:        config.Tools,
		memory:       config.Memory,
		state:        config.State,
		provider:     provider,
	}
}

func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Name:        "assistant",
		Model:       "gpt-4.1-mini",
		Temperature: 0.7,
		MaxTokens:   2000,
		State:       make(map[string]interface{}),
	}
}

func (a *agent) Chat(ctx context.Context, message string) (string, error) {
	if a.provider == nil {
		return "", fmt.Errorf("no LLM provider configured")
	}

	if a.memory != nil {
		if err := a.memory.Add(ctx, RoleUser, message); err != nil {
			return "", fmt.Errorf("failed to add message to memory: %w", err)
		}
	}

	messages, err := a.prepareMessages(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to prepare messages: %w", err)
	}

	var tools []llm.ToolDefinition
	if len(a.tools) > 0 {
		tools = a.convertToolsToLLMFormat()
	}

	req := llm.ChatRequest{
		Messages:    a.convertMessagesToLLM(messages),
		Model:       a.model,
		Temperature: a.temperature,
		MaxTokens:   a.maxTokens,
		Tools:       tools,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	choice := resp.Choices[0]

	if len(choice.Message.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, a.convertMessageFromLLM(choice.Message), messages)
	}

	if a.memory != nil {
		if err := a.memory.Add(ctx, RoleAssistant, choice.Message.Content); err != nil {
			return "", fmt.Errorf("failed to add response to memory: %w", err)
		}
	}

	return choice.Message.Content, nil
}

func (a *agent) WithTools(tools ...Tool) Agent {
	newAgent := *a
	newAgent.tools = append(newAgent.tools, tools...)
	return &newAgent
}

func (a *agent) WithMemory(memory Memory) Agent {
	newAgent := *a
	newAgent.memory = memory
	return &newAgent
}

func (a *agent) WithSystemPrompt(prompt string) Agent {
	newAgent := *a
	newAgent.systemPrompt = prompt
	return &newAgent
}

func (a *agent) WithTemperature(temp float64) Agent {
	newAgent := *a
	newAgent.temperature = temp
	return &newAgent
}

func (a *agent) WithMaxTokens(tokens int) Agent {
	newAgent := *a
	newAgent.maxTokens = tokens
	return &newAgent
}

func (a *agent) SetState(key string, value interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state[key] = value
}

func (a *agent) GetState(key string) interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state[key]
}

func (a *agent) ClearState() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = make(map[string]interface{})
}

func (a *agent) Name() string {
	return a.name
}

func (a *agent) Model() string {
	return a.model
}

func (a *agent) prepareMessages(ctx context.Context, currentMessage string) ([]ChatMessage, error) {
	var messages []ChatMessage
	if a.systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    RoleSystem,
			Content: a.systemPrompt,
		})
	}

	if a.memory != nil {
		history, err := a.memory.GetHistory(ctx, 10)
		if err == nil {
			for _, msg := range history {
				if msg.Role == RoleUser && msg.Content == currentMessage {
					continue
				}
				messages = append(messages, ChatMessage{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	if !a.messageExistsInHistory(messages, currentMessage) {
		messages = append(messages, ChatMessage{
			Role:    RoleUser,
			Content: currentMessage,
		})
	}

	return messages, nil
}

func (a *agent) messageExistsInHistory(messages []ChatMessage, content string) bool {
	for _, msg := range messages {
		if msg.Role == RoleUser && msg.Content == content {
			return true
		}
	}
	return false
}

func (a *agent) handleToolCalls(ctx context.Context, assistantMessage ChatMessage, conversationHistory []ChatMessage) (string, error) {
	if a.memory != nil {
		toolCallsJSON, _ := json.Marshal(assistantMessage.ToolCalls)
		if err := a.memory.Add(ctx, RoleAssistant, fmt.Sprintf("Tool calls: %s", string(toolCallsJSON))); err != nil {
			return "", fmt.Errorf("failed to add tool calls to memory: %w", err)
		}
	}

	var updatedMessages []ChatMessage
	updatedMessages = append(updatedMessages, conversationHistory...)
	updatedMessages = append(updatedMessages, assistantMessage)

	for _, toolCall := range assistantMessage.ToolCalls {
		if toolCall.Type != "function" {
			continue
		}

		var selectedTool Tool
		for _, tool := range a.tools {
			if tool.Name() == toolCall.Function.Name {
				selectedTool = tool
				break
			}
		}

		if selectedTool == nil {
			result := fmt.Sprintf("Error: Tool '%s' not found", toolCall.Function.Name)
			updatedMessages = append(updatedMessages, ChatMessage{
				Role:       RoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		var params map[string]interface{}
		if toolCall.Function.Arguments != "" {
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
			if err != nil {
				result := fmt.Sprintf("Error parsing tool parameters: %v", err)
				updatedMessages = append(updatedMessages, ChatMessage{
					Role:       RoleTool,
					Content:    result,
					ToolCallID: toolCall.ID,
				})
				continue
			}
		}

		toolResult, err := selectedTool.Execute(ctx, params)
		var result string
		if err != nil {
			result = fmt.Sprintf("Error executing tool: %v", err)
		} else {
			resultJSON, _ := json.Marshal(toolResult)
			result = string(resultJSON)
		}

		updatedMessages = append(updatedMessages, ChatMessage{
			Role:       RoleTool,
			Content:    result,
			ToolCallID: toolCall.ID,
		})

		if a.memory != nil {
			if err := a.memory.Add(ctx, RoleTool, fmt.Sprintf("Tool '%s' result: %s", selectedTool.Name(), result)); err != nil {
				return "", fmt.Errorf("failed to add tool result to memory: %w", err)
			}
		}
	}

	req := llm.ChatRequest{
		Messages:    a.convertMessagesToLLM(updatedMessages),
		Model:       a.model,
		Temperature: a.temperature,
		MaxTokens:   a.maxTokens,
	}

	resp, err := a.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call after tool execution failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned after tool execution")
	}

	finalResponse := resp.Choices[0].Message.Content

	if a.memory != nil {
		if err := a.memory.Add(ctx, RoleAssistant, finalResponse); err != nil {
			return "", fmt.Errorf("failed to add final response to memory: %w", err)
		}
	}

	return finalResponse, nil
}

func (a *agent) convertMessagesToLLM(messages []ChatMessage) []llm.Message {
	result := make([]llm.Message, len(messages))
	for i, msg := range messages {
		result[i] = llm.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			Name:       msg.Name,
			ToolCalls:  a.convertToolCallsToLLM(msg.ToolCalls),
			ToolCallID: msg.ToolCallID,
		}
	}
	return result
}

func (a *agent) convertMessageFromLLM(msg llm.Message) ChatMessage {
	return ChatMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		Name:       msg.Name,
		ToolCalls:  a.convertToolCallsFromLLM(msg.ToolCalls),
		ToolCallID: msg.ToolCallID,
	}
}

func (a *agent) convertToolCallsToLLM(toolCalls []ToolCall) []llm.ToolCall {
	result := make([]llm.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = llm.ToolCall{
			ID:   tc.ID,
			Type: tc.Type,
			Function: llm.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

func (a *agent) convertToolCallsFromLLM(toolCalls []llm.ToolCall) []ToolCall {
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

func (a *agent) convertToolsToLLMFormat() []llm.ToolDefinition {
	var tools []llm.ToolDefinition

	for _, tool := range a.tools {
		schema := tool.Schema()
		parameters := make(map[string]interface{})

		if schema != nil {
			parameters["type"] = schema.Type
			parameters["properties"] = schema.Properties
			parameters["required"] = schema.Required
		}

		tools = append(tools, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  parameters,
			},
		})
	}

	return tools
}
