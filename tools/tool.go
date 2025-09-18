package tools

import (
	"encoding/json"
	"time"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// Tool defines the tool interface
type Tool interface {
	Name() string
	Description() string
	Schema() *ToolSchema
	Execute(ctx runtime.Context, input json.RawMessage) (json.RawMessage, error)
	ExecuteAsync(ctx runtime.Context, input json.RawMessage) (<-chan ToolResult, error)
}

// ToolSchema describes the tool JSON schema
type ToolSchema struct {
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required"`
	Description string                 `json:"description"`
}

// ToolResult represents the outcome of a tool
type ToolResult struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// ToolConfig configures tool execution
type ToolConfig struct {
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	Sandbox    bool          `json:"sandbox"`
}

// DefaultToolConfig provides baseline configuration
var DefaultToolConfig = &ToolConfig{
	Timeout:    30 * time.Second,
	MaxRetries: 3,
	Sandbox:    true,
}

// BaseTool provides shared tool functionality
type BaseTool struct {
	name        string
	description string
	schema      *ToolSchema
	config      *ToolConfig
}

// NewBaseTool constructs a base tool
func NewBaseTool(name, description string, schema *ToolSchema) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
		config:      DefaultToolConfig,
	}
}

func (t *BaseTool) Name() string {
	return t.name
}

func (t *BaseTool) Description() string {
	return t.description
}

func (t *BaseTool) Schema() *ToolSchema {
	return t.schema
}

func (t *BaseTool) Config() *ToolConfig {
	return t.config
}

func (t *BaseTool) SetConfig(config *ToolConfig) {
	t.config = config
}

// Execute is a placeholder that should be overridden
func (t *BaseTool) Execute(ctx runtime.Context, input json.RawMessage) (json.RawMessage, error) {
	return nil, schema.NewToolError(t.name, "execute", schema.ErrToolExecutionFailed)
}

// ExecuteAsync executes the tool asynchronously
func (t *BaseTool) ExecuteAsync(ctx runtime.Context, input json.RawMessage) (<-chan ToolResult, error) {
	resultChan := make(chan ToolResult, 1)

	go func() {
		defer close(resultChan)

		result, err := t.Execute(ctx, input)
		if err != nil {
			resultChan <- ToolResult{
				Success: false,
				Error:   err.Error(),
			}
		} else {
			resultChan <- ToolResult{
				Success: true,
				Data:    result,
			}
		}
	}()

	return resultChan, nil
}

// ValidateInput performs lightweight validation
func (t *BaseTool) ValidateInput(input json.RawMessage) error {
	if t.schema == nil {
		return nil
	}

	// Basic JSON format validation
	var data map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return schema.NewValidationError("input", string(input), "invalid JSON format")
	}

	// Verify required fields
	for _, required := range t.schema.Required {
		if _, exists := data[required]; !exists {
			return schema.NewValidationError(required, nil, "required field missing")
		}
	}

	return nil
}

// CreateToolSchema builds a schema definition
func CreateToolSchema(description string, properties map[string]interface{}, required []string) *ToolSchema {
	return &ToolSchema{
		Type:        "object",
		Description: description,
		Properties:  properties,
		Required:    required,
	}
}

// StringProperty defines a string property
func StringProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

// NumberProperty defines a numeric property
func NumberProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "number",
		"description": description,
	}
}

// BooleanProperty defines a boolean property
func BooleanProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

// ArrayProperty defines an array property
func ArrayProperty(description string, itemType string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items": map[string]interface{}{
			"type": itemType,
		},
	}
}

// ObjectProperty defines an object property
func ObjectProperty(description string, properties map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}
}

// WithTimeout updates the timeout
func WithTimeout(timeout time.Duration) func(*ToolConfig) {
	return func(config *ToolConfig) {
		config.Timeout = timeout
	}
}

// WithMaxRetries updates the retry count
func WithMaxRetries(maxRetries int) func(*ToolConfig) {
	return func(config *ToolConfig) {
		config.MaxRetries = maxRetries
	}
}

// WithSandbox toggles sandboxing
func WithSandbox(sandbox bool) func(*ToolConfig) {
	return func(config *ToolConfig) {
		config.Sandbox = sandbox
	}
}

// NewCalculator provides a convenience constructor
func NewCalculator() Tool {
	// Importing the builtin package here would introduce a cycle,
	// so the actual implementation lives in builtin and we expose a stub here
	// Return nil for now; builtin holds the concrete implementation
	return nil
}
