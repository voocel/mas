package builtin

import (
	"encoding/json"
	"fmt"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// TransferTool enables agent handoffs
type TransferTool struct {
	*tools.BaseTool
	targetAgent string
}

// NewTransferTool constructs a handoff tool
func NewTransferTool(targetAgent, description string) *TransferTool {
	if description == "" {
		description = fmt.Sprintf("Transfer the conversation to %s agent", targetAgent)
	}

	schema := tools.CreateToolSchema(
		description,
		map[string]interface{}{
			"reason": tools.StringProperty("Reason for the transfer"),
			"priority": map[string]interface{}{
				"type":        "integer",
				"description": "Priority level (1-10, higher is more urgent)",
				"minimum":     1,
				"maximum":     10,
				"default":     5,
			},
			"context": tools.ObjectProperty("Additional context to pass to the target agent", nil),
		},
		[]string{}, // Reason is optional
	)

	baseTool := tools.NewBaseTool(fmt.Sprintf("transfer_to_%s", targetAgent), description, schema)

	return &TransferTool{
		BaseTool:    baseTool,
		targetAgent: targetAgent,
	}
}

func (t *TransferTool) Execute(ctx runtime.Context, input json.RawMessage) (json.RawMessage, error) {
	// Parse input arguments
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	// Create the handoff payload
	handoff := schema.NewHandoff(t.targetAgent)

	// Attach the reason
	if reason, ok := args["reason"].(string); ok && reason != "" {
		handoff.WithContext("reason", reason)
	}

	// Attach the priority
	if priority, ok := args["priority"].(float64); ok {
		handoff.WithPriority(int(priority))
	}

	// Attach additional context
	if contextData, ok := args["context"].(map[string]interface{}); ok {
		for k, v := range contextData {
			handoff.WithContext(k, v)
		}
	}

	// Persist the handoff in the runtime context
	ctx.SetStateValue(schema.HandoffPendingStateKey, handoff)

	result := fmt.Sprintf("Transferring to %s agent...", t.targetAgent)
	return json.Marshal(result)
}

// CreateTransferTools builds transfer tools for multiple targets
func CreateTransferTools(targets map[string]string) []tools.Tool {
	var transferTools []tools.Tool

	for targetAgent, description := range targets {
		transferTools = append(transferTools, NewTransferTool(targetAgent, description))
	}

	return transferTools
}

// TransferToExpert provides a reusable expert handoff tool
func TransferToExpert() tools.Tool {
	return NewTransferTool("expert", "Transfer to a domain expert when the task requires specialized knowledge")
}

// TransferToSupervisor hands off to a supervisor
func TransferToSupervisor() tools.Tool {
	return NewTransferTool("supervisor", "Transfer to supervisor for complex decisions or escalation")
}

// TransferToSpecialist hands off to a specialist
func TransferToSpecialist() tools.Tool {
	return NewTransferTool("specialist", "Transfer to a specialist for tasks requiring specific expertise")
}

// Predefined common transfer tools
var (
	// Frequently used transfer tools
	TransferToWriter     = NewTransferTool("writer", "Transfer to content writer for writing tasks")
	TransferToResearcher = NewTransferTool("researcher", "Transfer to researcher for research and analysis tasks")
	TransferToEngineer   = NewTransferTool("engineer", "Transfer to engineer for technical tasks")
	TransferToDesigner   = NewTransferTool("designer", "Transfer to designer for design-related tasks")
	TransferToSupport    = NewTransferTool("support", "Transfer to customer support for user assistance")
)

// GetCommonTransferTools returns the common transfer tool set
func GetCommonTransferTools() []tools.Tool {
	return []tools.Tool{
		TransferToWriter,
		TransferToResearcher,
		TransferToEngineer,
		TransferToDesigner,
		TransferToSupport,
		TransferToExpert(),
		TransferToSupervisor(),
	}
}

// TransferToolBuilder creates transfer tools via a builder pattern
type TransferToolBuilder struct {
	targetAgent string
	description string
	required    []string
	properties  map[string]interface{}
}

// NewTransferToolBuilder creates a transfer tool builder
func NewTransferToolBuilder(targetAgent string) *TransferToolBuilder {
	return &TransferToolBuilder{
		targetAgent: targetAgent,
		properties:  make(map[string]interface{}),
	}
}

// WithDescription overrides the description
func (b *TransferToolBuilder) WithDescription(description string) *TransferToolBuilder {
	b.description = description
	return b
}

// WithProperty adds a custom property
func (b *TransferToolBuilder) WithProperty(name string, property map[string]interface{}) *TransferToolBuilder {
	b.properties[name] = property
	return b
}

// WithRequired sets required parameters
func (b *TransferToolBuilder) WithRequired(required ...string) *TransferToolBuilder {
	b.required = required
	return b
}

// Build assembles the transfer tool
func (b *TransferToolBuilder) Build() tools.Tool {
	tool := NewTransferTool(b.targetAgent, b.description)

	// Create a custom tool when additional properties exist
	if len(b.properties) > 0 || len(b.required) > 0 {
		return &CustomTransferTool{
			TransferTool: tool,
			customProps:  b.properties,
			required:     b.required,
		}
	}

	return tool
}

// CustomTransferTool extends TransferTool
type CustomTransferTool struct {
	*TransferTool
	customProps map[string]interface{}
	required    []string
}

func (c *CustomTransferTool) Schema() *tools.ToolSchema {
	schema := c.TransferTool.Schema()

	// Merge custom properties
	if schema.Properties != nil {
		for k, v := range c.customProps {
			schema.Properties[k] = v
		}
	}

	// Apply required parameters
	if len(c.required) > 0 {
		schema.Required = c.required
	}

	return schema
}

// ExtractHandoffFromToolCall derives a handoff from the tool call
func ExtractHandoffFromToolCall(toolCall schema.ToolCall) (*schema.Handoff, error) {
	if !IsTransferTool(toolCall.Name) {
		return nil, fmt.Errorf("not a transfer tool: %s", toolCall.Name)
	}

	// Extract the target agent
	target := ExtractTargetFromToolName(toolCall.Name)
	if target == "" {
		return nil, fmt.Errorf("cannot extract target from tool name: %s", toolCall.Name)
	}

	// Parse arguments
	var args map[string]interface{}
	if err := json.Unmarshal(toolCall.Args, &args); err != nil {
		// On failure, fall back to a basic handoff
		return schema.NewHandoff(target).WithContext("reason", "function_call"), nil
	}

	handoff := schema.NewHandoff(target)

	// Pull information from arguments
	if reason, ok := args["reason"].(string); ok {
		handoff.WithContext("reason", reason)
	}
	if priority, ok := args["priority"].(float64); ok {
		handoff.WithPriority(int(priority))
	}
	if contextData, ok := args["context"].(map[string]interface{}); ok {
		for k, v := range contextData {
			handoff.WithContext(k, v)
		}
	}

	return handoff, nil
}

// IsTransferTool reports whether the tool is a transfer tool
func IsTransferTool(toolName string) bool {
	return len(toolName) > 12 && toolName[:12] == "transfer_to_"
}

// ExtractTargetFromToolName pulls the target from the tool name
func ExtractTargetFromToolName(toolName string) string {
	if !IsTransferTool(toolName) {
		return ""
	}
	return toolName[12:] // Remove the "transfer_to_" prefix
}
