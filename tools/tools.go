package tools

import (
	"context"
	"encoding/json"
)

// Tool defines a tool that agents can use
type Tool interface {
	// Name returns the tool's name
	Name() string

	// Description returns the tool's description
	Description() string

	// ParameterSchema returns the tool's parameter JSON Schema
	ParameterSchema() json.RawMessage

	// Execute runs the tool and returns the result
	Execute(ctx context.Context, params json.RawMessage) (interface{}, error)
}

// Registry is a tool registry
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register registers a tool
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List lists all available tools
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// BaseTool provides a base implementation of the Tool interface
type BaseTool struct {
	name        string
	description string
	schema      json.RawMessage
	handler     func(ctx context.Context, params json.RawMessage) (interface{}, error)
}

// NewBaseTool creates a new base tool
func NewBaseTool(
	name string,
	description string,
	schema json.RawMessage,
	handler func(ctx context.Context, params json.RawMessage) (interface{}, error),
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

func (t *BaseTool) ParameterSchema() json.RawMessage {
	return t.schema
}

func (t *BaseTool) Execute(ctx context.Context, params json.RawMessage) (interface{}, error) {
	return t.handler(ctx, params)
}

var (
	ErrToolNotFound      = ToolError{Code: "tool_not_found", Message: "Tool not found"}
	ErrInvalidParameters = ToolError{Code: "invalid_parameters", Message: "Invalid parameters"}
	ErrExecutionFailed   = ToolError{Code: "execution_failed", Message: "Tool execution failed"}
)

type ToolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e ToolError) Error() string {
	return e.Message
}

func (e ToolError) WithDetails(details string) ToolError {
	e.Details = details
	return e
}
