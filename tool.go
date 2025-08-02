package mas

import (
	"context"
	"encoding/json"
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

// ToolSchema defines the parameter schema for a tool
type ToolSchema struct {
	Type        string                     `json:"type"`
	Properties  map[string]*PropertySchema `json:"properties"`
	Required    []string                   `json:"required"`
	Description string                     `json:"description,omitempty"`
}

// PropertySchema defines a property in the tool schema
type PropertySchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Items       *PropertySchema `json:"items,omitempty"`
}

// ToolExecutor is a function that executes a tool
type ToolExecutor func(ctx context.Context, params map[string]any) (any, error)

// BaseTool provides a basic implementation of the Tool interface
type BaseTool struct {
	name        string
	description string
	schema      *ToolSchema
	executor    ToolExecutor
}

// NewTool creates a new tool with the given configuration
func NewTool(name, description string, schema *ToolSchema, executor ToolExecutor) Tool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
		executor:    executor,
	}
}

// Name returns the tool's name
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool's description
func (t *BaseTool) Description() string {
	return t.description
}

// Execute runs the tool with the given parameters
func (t *BaseTool) Execute(ctx context.Context, params map[string]any) (any, error) {
	return t.executor(ctx, params)
}

// Schema returns the tool's parameter schema
func (t *BaseTool) Schema() *ToolSchema {
	return t.schema
}

// NewSimpleTool creates a simple tool with minimal configuration
func NewSimpleTool(name, description string, executor ToolExecutor) Tool {
	return &BaseTool{
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

// Helper functions for creating common property schemas

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

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
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

// ToJSON converts a tool result to JSON string
func (r *ToolResult) ToJSON() string {
	data, _ := json.Marshal(r)
	return string(data)
}

// ToolRegistry manages a collection of tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tools
func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Names returns all tool names
func (r *ToolRegistry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}