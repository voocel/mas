package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolAdapter adapts various custom tools to the framework's Tool interface
type ToolAdapter struct {
	name         string
	description  string
	schema       json.RawMessage
	internalTool interface{} // actual tool implementation
}

// NewToolAdapter creates a new tool adapter
func NewToolAdapter(name, description string, schema json.RawMessage, tool interface{}) *ToolAdapter {
	return &ToolAdapter{
		name:         name,
		description:  description,
		schema:       schema,
		internalTool: tool,
	}
}

// Name returns the tool name
func (a *ToolAdapter) Name() string {
	if t, ok := a.internalTool.(interface{ Name() string }); ok {
		return t.Name()
	}
	if t, ok := a.internalTool.(interface{ GetName() string }); ok {
		return t.GetName()
	}
	return a.name
}

// Description returns the tool description
func (a *ToolAdapter) Description() string {
	if t, ok := a.internalTool.(interface{ Description() string }); ok {
		return t.Description()
	}
	if t, ok := a.internalTool.(interface{ GetDescription() string }); ok {
		return t.GetDescription()
	}
	return a.description
}

// ParameterSchema returns the tool's parameter schema
func (a *ToolAdapter) ParameterSchema() json.RawMessage {
	if t, ok := a.internalTool.(interface{ ParameterSchema() json.RawMessage }); ok {
		return t.ParameterSchema()
	}
	if t, ok := a.internalTool.(interface{ Schema() json.RawMessage }); ok {
		return t.Schema()
	}
	return a.schema
}

// Execute runs the tool
func (a *ToolAdapter) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	if t, ok := a.internalTool.(interface {
		Execute(context.Context, json.RawMessage) (interface{}, error)
	}); ok {
		return t.Execute(ctx, params)
	}

	var paramMap map[string]interface{}
	if err := json.Unmarshal(params, &paramMap); err != nil {
		return nil, fmt.Errorf("failed to parse parameters: %w", err)
	}

	if t, ok := a.internalTool.(interface {
		Execute(context.Context, map[string]interface{}) (interface{}, error)
	}); ok {
		return t.Execute(ctx, paramMap)
	}

		return nil, fmt.Errorf("unsupported tool type")
}

// AdaptTool is a convenience method to adapt custom tools to the framework's standard Tool interface
func AdaptTool(name, description string, tool interface{}) Tool {
	defaultSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "query parameter"
			}
		},
		"required": ["query"]
	}`)

	return NewToolAdapter(name, description, defaultSchema, tool)
}
