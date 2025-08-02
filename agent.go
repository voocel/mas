package mas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voocel/mas/llm"
)

// Agent represents an intelligent agent that can chat, use tools, and maintain memory
type Agent interface {
	// Core chat functionality
	Chat(ctx context.Context, message string) (string, error)

	// Fluent configuration methods
	WithTools(tools ...Tool) Agent
	WithMemory(memory Memory) Agent
	WithSystemPrompt(prompt string) Agent
	WithTemperature(temp float64) Agent
	WithMaxTokens(tokens int) Agent

	// State management
	SetState(key string, value interface{})
	GetState(key string) interface{}
	ClearState()

	// Information methods
	Name() string
	Model() string
}

// AgentConfig contains configuration for creating an agent
type AgentConfig struct {
	Name         string
	Model        string
	APIKey       string
	SystemPrompt string
	Temperature  float64
	MaxTokens    int
	Tools        []Tool
	Memory       Memory
	State        map[string]interface{}
	Provider     llm.Provider
}

// DefaultAgentConfig returns a default configuration
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Name:        "assistant",
		Model:       "gpt-4.1-mini",
		Temperature: 0.7,
		MaxTokens:   2000,
		State:       make(map[string]interface{}),
	}
}

// NewAgent creates a new agent with minimal configuration
func NewAgent(model, apiKey string) Agent {
	config := DefaultAgentConfig()
	config.Model = model
	config.APIKey = apiKey

	// Create LLM provider
	provider, err := llm.NewProvider(model, apiKey)
	if err != nil {
		// For now, we'll create a fallback provider
		// In production, this should be handled more gracefully
		provider = nil
	}
	config.Provider = provider

	return NewAgentWithConfig(config)
}

// NewAgentWithConfig creates a new agent with full configuration
func NewAgentWithConfig(config AgentConfig) Agent {
	if config.State == nil {
		config.State = make(map[string]interface{})
	}

	// Create provider if not provided
	if config.Provider == nil && config.APIKey != "" {
		provider, err := llm.NewProvider(config.Model, config.APIKey)
		if err == nil {
			config.Provider = provider
		}
	}

	return &agent{
		config: config,
	}
}

// agent is the default implementation of the Agent interface
type agent struct {
	config AgentConfig
}

// Chat implements the core chat functionality
func (a *agent) Chat(ctx context.Context, message string) (string, error) {
	if a.config.Provider == nil {
		return "", fmt.Errorf("no LLM provider configured")
	}

	// Add user message to memory if available
	if a.config.Memory != nil {
		err := a.config.Memory.Add(ctx, RoleUser, message)
		if err != nil {
			return "", fmt.Errorf("failed to add message to memory: %w", err)
		}
	}

	// Prepare messages for LLM
	messages, err := a.prepareMessages(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to prepare messages: %w", err)
	}

	// Prepare tools for function calling if available
	var tools []llm.ToolDefinition
	if len(a.config.Tools) > 0 {
		tools = a.convertToolsToLLMFormat()
	}

	// Create chat request
	req := llm.ChatRequest{
		Messages:    messages,
		Model:       a.config.Model,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
		Tools:       tools,
	}

	// Call LLM
	response, err := a.config.Provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	choice := response.Choices[0]

	// Handle tool calls if present
	if len(choice.Message.ToolCalls) > 0 {
		return a.handleToolCalls(ctx, choice.Message, messages)
	}

	// Add assistant response to memory
	if a.config.Memory != nil {
		err := a.config.Memory.Add(ctx, RoleAssistant, choice.Message.Content)
		if err != nil {
			return "", fmt.Errorf("failed to add response to memory: %w", err)
		}
	}

	return choice.Message.Content, nil
}

// Fluent configuration methods
func (a *agent) WithTools(tools ...Tool) Agent {
	newConfig := a.config
	newConfig.Tools = append(newConfig.Tools, tools...)
	return &agent{config: newConfig}
}

func (a *agent) WithMemory(memory Memory) Agent {
	newConfig := a.config
	newConfig.Memory = memory
	return &agent{config: newConfig}
}

func (a *agent) WithSystemPrompt(prompt string) Agent {
	newConfig := a.config
	newConfig.SystemPrompt = prompt
	return &agent{config: newConfig}
}

func (a *agent) WithTemperature(temp float64) Agent {
	newConfig := a.config
	newConfig.Temperature = temp
	return &agent{config: newConfig}
}

func (a *agent) WithMaxTokens(tokens int) Agent {
	newConfig := a.config
	newConfig.MaxTokens = tokens
	return &agent{config: newConfig}
}

// State management
func (a *agent) SetState(key string, value interface{}) {
	a.config.State[key] = value
}

func (a *agent) GetState(key string) interface{} {
	return a.config.State[key]
}

func (a *agent) ClearState() {
	a.config.State = make(map[string]interface{})
}

func (a *agent) Name() string {
	return a.config.Name
}

func (a *agent) Model() string {
	return a.config.Model
}

// prepareMessages prepares the message list for LLM API call
func (a *agent) prepareMessages(ctx context.Context, currentMessage string) ([]llm.Message, error) {
	var messages []llm.Message

	// Add system prompt if available
	if a.config.SystemPrompt != "" {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: a.config.SystemPrompt,
		})
	}

	// Add conversation history from memory
	if a.config.Memory != nil {
		history, err := a.config.Memory.GetHistory(ctx, 10) // Get last 10 messages
		if err == nil {
			for _, msg := range history {
				// Skip the current message if it's already in history
				if msg.Role == RoleUser && msg.Content == currentMessage {
					continue
				}
				messages = append(messages, llm.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	// Add current message if not already added from memory
	if !a.messageExistsInHistory(messages, currentMessage) {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: currentMessage,
		})
	}

	return messages, nil
}

// messageExistsInHistory checks if a message already exists in the message history
func (a *agent) messageExistsInHistory(messages []llm.Message, content string) bool {
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content == content {
			return true
		}
	}
	return false
}

// convertToolsToLLMFormat converts internal tools to LLM API format
func (a *agent) convertToolsToLLMFormat() []llm.ToolDefinition {
	var tools []llm.ToolDefinition

	for _, tool := range a.config.Tools {
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

// handleToolCalls processes tool calls from LLM response
func (a *agent) handleToolCalls(ctx context.Context, assistantMessage llm.Message, conversationHistory []llm.Message) (string, error) {
	// Add assistant message with tool calls to memory
	if a.config.Memory != nil {
		toolCallsJSON, _ := json.Marshal(assistantMessage.ToolCalls)
		err := a.config.Memory.Add(ctx, RoleAssistant, fmt.Sprintf("Tool calls: %s", string(toolCallsJSON)))
		if err != nil {
			return "", fmt.Errorf("failed to add tool calls to memory: %w", err)
		}
	}

	var toolResults []string
	var updatedMessages []llm.Message
	updatedMessages = append(updatedMessages, conversationHistory...)
	updatedMessages = append(updatedMessages, assistantMessage)

	// Execute each tool call
	for _, toolCall := range assistantMessage.ToolCalls {
		if toolCall.Type != "function" {
			continue
		}

		var selectedTool Tool
		for _, tool := range a.config.Tools {
			if tool.Name() == toolCall.Function.Name {
				selectedTool = tool
				break
			}
		}

		if selectedTool == nil {
			result := fmt.Sprintf("Error: Tool '%s' not found", toolCall.Function.Name)
			toolResults = append(toolResults, result)

			// Add tool result message
			updatedMessages = append(updatedMessages, llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: toolCall.ID,
			})
			continue
		}

		// Parse tool parameters
		var params map[string]interface{}
		if toolCall.Function.Arguments != "" {
			err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params)
			if err != nil {
				result := fmt.Sprintf("Error parsing tool parameters: %v", err)
				toolResults = append(toolResults, result)

				updatedMessages = append(updatedMessages, llm.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: toolCall.ID,
				})
				continue
			}
		}

		// Execute the tool
		toolResult, err := selectedTool.Execute(ctx, params)
		var result string
		if err != nil {
			result = fmt.Sprintf("Error executing tool: %v", err)
		} else {
			resultJSON, _ := json.Marshal(toolResult)
			result = string(resultJSON)
		}

		toolResults = append(toolResults, result)

		// Add tool result message
		updatedMessages = append(updatedMessages, llm.Message{
			Role:       "tool",
			Content:    result,
			ToolCallID: toolCall.ID,
		})

		// Add tool result to memory
		if a.config.Memory != nil {
			err := a.config.Memory.Add(ctx, RoleTool, fmt.Sprintf("Tool '%s' result: %s", selectedTool.Name(), result))
			if err != nil {
				return "", fmt.Errorf("failed to add tool result to memory: %w", err)
			}
		}
	}

	// Make another LLM call with tool results
	req := llm.ChatRequest{
		Messages:    updatedMessages,
		Model:       a.config.Model,
		Temperature: a.config.Temperature,
		MaxTokens:   a.config.MaxTokens,
	}

	response, err := a.config.Provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM call after tool execution failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned after tool execution")
	}

	finalResponse := response.Choices[0].Message.Content

	// Add final response to memory
	if a.config.Memory != nil {
		err := a.config.Memory.Add(ctx, RoleAssistant, finalResponse)
		if err != nil {
			return "", fmt.Errorf("failed to add final response to memory: %w", err)
		}
	}

	return finalResponse, nil
}
