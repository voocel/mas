package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/voocel/mas/schema"
)

// Tool defines the tool interface.
type Tool interface {
	Name() string
	Description() string
	Schema() *ToolSchema
	Capabilities() []Capability
	Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
	ExecuteAsync(ctx context.Context, input json.RawMessage) (<-chan ToolResult, error)
}

// Capability defines tool side effects.
type Capability string

const (
	CapabilityNetwork Capability = "network"
	CapabilityFile    Capability = "file"
	CapabilityUnsafe  Capability = "unsafe"
)

// ToolSchema describes a tool JSON schema.
type ToolSchema struct {
	Type        string                 `json:"type"`
	Properties  map[string]interface{} `json:"properties"`
	Required    []string               `json:"required"`
	Description string                 `json:"description"`
}

// ToolResult represents a tool execution result.
type ToolResult struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// ToolConfig configures tool execution.
type ToolConfig struct {
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	Sandbox    bool          `json:"sandbox"`
}

// DefaultToolConfig provides default configuration.
var DefaultToolConfig = &ToolConfig{
	Timeout:    30 * time.Second,
	MaxRetries: 3,
	Sandbox:    true,
}

// BaseTool provides shared tool functionality.
type BaseTool struct {
	name        string
	description string
	schema      *ToolSchema
	config      *ToolConfig
	caps        []Capability
}

// NewBaseTool creates a base tool.
func NewBaseTool(name, description string, schema *ToolSchema) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		schema:      schema,
		config:      cloneToolConfig(DefaultToolConfig),
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

func (t *BaseTool) Capabilities() []Capability {
	return append([]Capability(nil), t.caps...)
}

func (t *BaseTool) Config() *ToolConfig {
	if t.config == nil {
		t.config = cloneToolConfig(DefaultToolConfig)
	}
	return t.config
}

func (t *BaseTool) SetConfig(config *ToolConfig) {
	t.config = cloneToolConfig(config)
}

// Execute is a default implementation and should be overridden.
func (t *BaseTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return nil, schema.NewToolError(t.name, "execute", schema.ErrToolExecutionFailed)
}

// ExecuteAsync executes the tool asynchronously.
func (t *BaseTool) ExecuteAsync(ctx context.Context, input json.RawMessage) (<-chan ToolResult, error) {
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

// WithCapabilities sets capability markers.
func (t *BaseTool) WithCapabilities(caps ...Capability) *BaseTool {
	t.caps = append([]Capability(nil), caps...)
	return t
}

// ValidateInput performs lightweight validation.
func (t *BaseTool) ValidateInput(input json.RawMessage) error {
	if t.schema == nil {
		return nil
	}

	// Basic JSON format validation
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		if len(t.schema.Required) == 0 {
			return nil
		}
		return schema.NewValidationError("input", string(input), "required field missing")
	}
	var data map[string]interface{}
	if err := json.Unmarshal(trimmed, &data); err != nil {
		return schema.NewValidationError("input", string(trimmed), "invalid JSON format")
	}
	if data == nil {
		data = map[string]interface{}{}
	}

	// Verify required fields
	for _, required := range t.schema.Required {
		if _, exists := data[required]; !exists {
			return schema.NewValidationError(required, nil, "required field missing")
		}
	}

	return nil
}

// CreateToolSchema builds a schema.
func CreateToolSchema(description string, properties map[string]interface{}, required []string) *ToolSchema {
	return &ToolSchema{
		Type:        "object",
		Description: description,
		Properties:  properties,
		Required:    required,
	}
}

// StringProperty defines a string property.
func StringProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

// NumberProperty defines a numeric property.
func NumberProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "number",
		"description": description,
	}
}

// BooleanProperty defines a boolean property.
func BooleanProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "boolean",
		"description": description,
	}
}

// ArrayProperty defines an array property.
func ArrayProperty(description string, itemType string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"description": description,
		"items": map[string]interface{}{
			"type": itemType,
		},
	}
}

func cloneToolConfig(cfg *ToolConfig) *ToolConfig {
	if cfg == nil {
		if DefaultToolConfig == nil {
			return &ToolConfig{}
		}
		copyCfg := *DefaultToolConfig
		return &copyCfg
	}
	copyCfg := *cfg
	return &copyCfg
}

// ObjectProperty defines an object property.
func ObjectProperty(description string, properties map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type":        "object",
		"description": description,
		"properties":  properties,
	}
}

// WithTimeout sets a timeout.
func WithTimeout(timeout time.Duration) func(*ToolConfig) {
	return func(config *ToolConfig) {
		config.Timeout = timeout
	}
}

// WithMaxRetries sets max retries.
func WithMaxRetries(maxRetries int) func(*ToolConfig) {
	return func(config *ToolConfig) {
		config.MaxRetries = maxRetries
	}
}

// WithSandbox toggles sandboxing (reserved).
func WithSandbox(sandbox bool) func(*ToolConfig) {
	return func(config *ToolConfig) {
		config.Sandbox = sandbox
	}
}

// ToolFunction defines a tool function signature.
// The function must be: func(ctx context.Context, ...args) (string, error)
type ToolFunction interface{}

// FunctionTool wraps a function as a tool.
type FunctionTool struct {
	name        string
	description string
	fn          ToolFunction
	schema      *ToolSchema
	caps        []Capability
}

// NewFunctionTool creates a tool from a function.
func NewFunctionTool(name, description string, fn ToolFunction) (*FunctionTool, error) {
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("tool must be a function")
	}

	if fnType.NumOut() != 2 {
		return nil, fmt.Errorf("function must return (string, error)")
	}

	if fnType.Out(0) != reflect.TypeOf("") {
		return nil, fmt.Errorf("function must return string as first value")
	}

	if !fnType.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		return nil, fmt.Errorf("function must return error as second value")
	}

	if fnType.NumIn() < 1 || fnType.In(0) != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return nil, fmt.Errorf("function must take context.Context as first parameter")
	}

	schema := generateSchemaFromFunction(fnType)

	return &FunctionTool{
		name:        name,
		description: description,
		fn:          fn,
		schema:      schema,
	}, nil
}

func (ft *FunctionTool) Name() string {
	return ft.name
}

func (ft *FunctionTool) Description() string {
	return ft.description
}

func (ft *FunctionTool) Schema() *ToolSchema {
	return ft.schema
}

func (ft *FunctionTool) Capabilities() []Capability {
	return append([]Capability(nil), ft.caps...)
}

func (ft *FunctionTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	fnValue := reflect.ValueOf(ft.fn)
	fnType := fnValue.Type()

	var params map[string]interface{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &params); err != nil {
			return nil, fmt.Errorf("invalid input JSON: %v", err)
		}
	} else {
		params = make(map[string]interface{})
	}

	args := []reflect.Value{reflect.ValueOf(ctx)}

	for i := 1; i < fnType.NumIn(); i++ {
		paramName := fmt.Sprintf("param%d", i-1)
		paramType := fnType.In(i)

		var argValue reflect.Value
		if val, exists := params[paramName]; exists {
			argValue = reflect.ValueOf(val)
			if argValue.Type() != paramType {
				if argValue.Type().ConvertibleTo(paramType) {
					argValue = argValue.Convert(paramType)
				} else {
					return nil, fmt.Errorf("parameter %s: cannot convert %v to %v", paramName, argValue.Type(), paramType)
				}
			}
		} else {
			argValue = reflect.Zero(paramType)
		}

		args = append(args, argValue)
	}

	results := fnValue.Call(args)

	if !results[1].IsNil() {
		return nil, results[1].Interface().(error)
	}

	result := results[0].String()
	return json.Marshal(map[string]string{"result": result})
}

func (ft *FunctionTool) ExecuteAsync(ctx context.Context, input json.RawMessage) (<-chan ToolResult, error) {
	resultChan := make(chan ToolResult, 1)

	go func() {
		defer close(resultChan)

		result, err := ft.Execute(ctx, input)
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

// WithCapabilities sets capability markers.
func (ft *FunctionTool) WithCapabilities(caps ...Capability) *FunctionTool {
	ft.caps = append([]Capability(nil), caps...)
	return ft
}

// generateSchemaFromFunction builds a schema from a function signature.
func generateSchemaFromFunction(fnType reflect.Type) *ToolSchema {
	properties := make(map[string]interface{})
	required := make([]string, 0)

	// Analyze parameter types and generate a schema (skip the first Context parameter)
	for i := 1; i < fnType.NumIn(); i++ {
		paramType := fnType.In(i)
		paramName := fmt.Sprintf("param%d", i-1)

		switch paramType.Kind() {
		case reflect.String:
			properties[paramName] = map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("String parameter %d", i-1),
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			properties[paramName] = map[string]interface{}{
				"type":        "integer",
				"description": fmt.Sprintf("Integer parameter %d", i-1),
			}
		case reflect.Float32, reflect.Float64:
			properties[paramName] = map[string]interface{}{
				"type":        "number",
				"description": fmt.Sprintf("Number parameter %d", i-1),
			}
		case reflect.Bool:
			properties[paramName] = map[string]interface{}{
				"type":        "boolean",
				"description": fmt.Sprintf("Boolean parameter %d", i-1),
			}
		default:
			properties[paramName] = map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Parameter %d (converted to string)", i-1),
			}
		}

		required = append(required, paramName)
	}

	return &ToolSchema{
		Type:        "object",
		Properties:  properties,
		Required:    required,
		Description: "Auto-generated schema from function signature",
	}
}
