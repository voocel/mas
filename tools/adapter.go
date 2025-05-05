package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolAdapter provides a way to adapt different tool implementations to the standard Tool interface
type ToolAdapter struct {
	name         string
	description  string
	schema       json.RawMessage
	internalTool interface{} // actual tool implementation
}

// NewToolAdapter creates a new tool adapter
func NewToolAdapter(name, description string, schema json.RawMessage, tool interface{}) Tool {
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

// Schema returns the tool parameter schema
func (a *ToolAdapter) Schema() json.RawMessage {
	if t, ok := a.internalTool.(interface{ Schema() json.RawMessage }); ok {
		return t.Schema()
	}
	return a.schema
}

// Execute executes the tool
func (a *ToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Try to use the new interface directly
	if t, ok := a.internalTool.(interface {
		Execute(context.Context, map[string]interface{}) (interface{}, error)
	}); ok {
		return t.Execute(ctx, params)
	}

	// Try to call custom method
	if t, ok := a.internalTool.(interface {
		Run(context.Context, map[string]interface{}) (interface{}, error)
	}); ok {
		return t.Run(ctx, params)
	}

	// Try to call function-style tool
	if fn, ok := a.internalTool.(func(context.Context, map[string]interface{}) (interface{}, error)); ok {
		return fn(ctx, params)
	}

	return nil, fmt.Errorf("unsupported tool type")
}
