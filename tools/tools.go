package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Tool defines a tool that agents can use
type Tool interface {
	// Name returns the tool's name
	Name() string

	// Description returns the tool's description
	Description() string

	// Schema returns the tool's parameter schema
	Schema() json.RawMessage

	// Execute runs the tool with given parameters and returns the result
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// BaseTool provides a base implementation of the Tool interface
type BaseTool struct {
	name        string
	description string
	schema      json.RawMessage
	handler     func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// NewTool creates a new tool with the given name, description, and execution handler
func NewTool(
	name string,
	description string,
	schema json.RawMessage,
	handler func(ctx context.Context, params map[string]interface{}) (interface{}, error),
) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	}
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) Schema() json.RawMessage {
	return t.schema
}

func (t *BaseTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return t.handler(ctx, params)
}

// Common error types
var (
	ErrToolNotFound      = fmt.Errorf("tool not found")
	ErrInvalidParameters = fmt.Errorf("invalid parameters")
	ErrExecutionFailed   = fmt.Errorf("tool execution failed")
)

// Helper function to convert JSON parameters to map
func ParseParams(paramsJSON json.RawMessage) (map[string]interface{}, error) {
	var params map[string]interface{}
	if err := json.Unmarshal(paramsJSON, &params); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}
	return params, nil
}

// Helper function to execute a tool with JSON parameters
func ExecuteWithJSON(ctx context.Context, tool Tool, paramsJSON json.RawMessage) (interface{}, error) {
	params, err := ParseParams(paramsJSON)
	if err != nil {
		return nil, err
	}
	return tool.Execute(ctx, params)
}

// NewRawSchema creates a raw JSON schema from a string
func NewRawSchema(schema string) json.RawMessage {
	return json.RawMessage(schema)
}

// Generate OpenAI Function format description for a tool
func GenerateFunctionDescription(tool Tool) map[string]interface{} {
	var schema map[string]interface{}
	_ = json.Unmarshal(tool.Schema(), &schema)

	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  schema,
		},
	}
}

// Convert a list of tools to OpenAI Functions format
func ConvertToolsToFunctions(tools []Tool) []map[string]interface{} {
	functions := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		functions[i] = GenerateFunctionDescription(tool)
	}
	return functions
}
