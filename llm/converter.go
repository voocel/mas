package llm

import (
	"github.com/voocel/litellm"
)

// Helper functions for converting between formats

// convertMessagesToLiteLLM converts our message format to litellm format
func convertMessagesToLiteLLM(messages []Message) []litellm.Message {
	result := make([]litellm.Message, len(messages))
	for i, msg := range messages {
		result[i] = litellm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// convertToolsToLiteLLM converts our tool format to litellm format
func convertToolsToLiteLLM(tools []ToolDefinition) []litellm.Tool {
	if len(tools) == 0 {
		return nil
	}

	result := make([]litellm.Tool, len(tools))
	for i, tool := range tools {
		result[i] = litellm.Tool{
			Type: tool.Type,
			Function: litellm.FunctionSchema{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	return result
}

// convertToolCallsFromLiteLLM converts litellm tool calls to our format
func convertToolCallsFromLiteLLM(toolCalls []litellm.ToolCall) []ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

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