package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/voocel/mas/llm"
)

// Role constants for messages
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// prepareMessages prepares the message list for LLM API call
func (a *agent) prepareMessages(ctx context.Context, currentMessage string) ([]llm.Message, error) {
	var messages []llm.Message

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
