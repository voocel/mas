package mas

import (
	"context"
)

// Tool represents a tool that agents can use
type Tool interface {
	// Name returns the tool's name
	Name() string

	// Description returns what the tool does
	Description() string

	// Execute runs the tool with given parameters
	Execute(ctx context.Context, params map[string]any) (any, error)

	// Schema returns the JSON schema for the tool's parameters
	Schema() *ToolSchema
}

// ToolExecutor is a function that executes a tool
type ToolExecutor func(ctx context.Context, params map[string]any) (any, error)

// NewTool creates a new tool with the given configuration
func NewTool(name, description string, schema *ToolSchema, executor ToolExecutor) Tool {
	return &baseTool{
		name:        name,
		description: description,
		schema:      schema,
		executor:    executor,
	}
}

// NewSimpleTool creates a simple tool with minimal configuration
func NewSimpleTool(name, description string, executor ToolExecutor) Tool {
	return &baseTool{
		name:        name,
		description: description,
		schema: &ToolSchema{
			Type:       "object",
			Properties: make(map[string]*PropertySchema),
			Required:   []string{},
		},
		executor: executor,
	}
}

// baseTool provides a basic implementation of the Tool interface
type baseTool struct {
	name        string
	description string
	schema      *ToolSchema
	executor    ToolExecutor
}

func (t *baseTool) Name() string {
	return t.name
}

func (t *baseTool) Description() string {
	return t.description
}

func (t *baseTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.executor(ctx, params)
}

func (t *baseTool) Schema() *ToolSchema {
	return t.schema
}

// StringProperty creates a string property schema
func StringProperty(description string) *PropertySchema {
	return &PropertySchema{
		Type:        "string",
		Description: description,
	}
}

// NumberProperty creates a number property schema
func NumberProperty(description string) *PropertySchema {
	return &PropertySchema{
		Type:        "number",
		Description: description,
	}
}

// BooleanProperty creates a boolean property schema
func BooleanProperty(description string) *PropertySchema {
	return &PropertySchema{
		Type:        "boolean",
		Description: description,
	}
}

// ArrayProperty creates an array property schema
func ArrayProperty(description string, items *PropertySchema) *PropertySchema {
	return &PropertySchema{
		Type:        "array",
		Description: description,
		Items:       items,
	}
}

// EnumProperty creates an enum property schema
func EnumProperty(description string, values []string) *PropertySchema {
	return &PropertySchema{
		Type:        "string",
		Description: description,
		Enum:        values,
	}
}

// NewSuccessResult creates a successful tool result
func NewSuccessResult(data interface{}) *ToolResult {
	return &ToolResult{
		Success: true,
		Data:    data,
	}
}

// NewErrorResult creates an error tool result
func NewErrorResult(err error) *ToolResult {
	return &ToolResult{
		Success: false,
		Error:   err.Error(),
	}
}
