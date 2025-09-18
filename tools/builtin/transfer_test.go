package builtin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

func TestTransferTool_Creation(t *testing.T) {
	tool := NewTransferTool("writer", "Transfer to content writer")
	
	if tool.Name() != "transfer_to_writer" {
		t.Errorf("Expected name 'transfer_to_writer', got '%s'", tool.Name())
	}
	
	if tool.Description() != "Transfer to content writer" {
		t.Errorf("Expected description 'Transfer to content writer', got '%s'", tool.Description())
	}
	
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Schema should not be nil")
	}
	
	if schema.Type != "object" {
		t.Errorf("Expected schema type 'object', got '%s'", schema.Type)
	}
}

func TestTransferTool_Execute(t *testing.T) {
	tool := NewTransferTool("engineer", "Transfer to engineer")
	
	// Create a test context
	ctx := runtime.NewContext(context.Background(), "test-session", "test-trace")
	
	// Prepare arguments
	args := map[string]interface{}{
		"reason":   "Need technical expertise",
		"priority": 8,
		"context": map[string]interface{}{
			"task_type": "debugging",
		},
	}
	
	input, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal args: %v", err)
	}
	
	// Execute the tool
	result, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	// Verify the result
	var resultStr string
	if err := json.Unmarshal(result, &resultStr); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	expected := "Transferring to engineer agent..."
	if resultStr != expected {
		t.Errorf("Expected result '%s', got '%s'", expected, resultStr)
	}
	
	// Ensure the handoff is stored in context
	handoffValue, exists := ctx.State().Get("pending_handoff")
	if !exists {
		t.Fatal("Expected pending_handoff to be set in context")
	}
	
	handoff, ok := handoffValue.(*schema.Handoff)
	if !ok {
		t.Fatal("Expected pending_handoff to be *schema.Handoff")
	}
	
	if handoff.Target != "engineer" {
		t.Errorf("Expected handoff target 'engineer', got '%s'", handoff.Target)
	}
	
	if handoff.Priority != 8 {
		t.Errorf("Expected handoff priority 8, got %d", handoff.Priority)
	}
	
	reason, exists := handoff.GetContext("reason")
	if !exists || reason != "Need technical expertise" {
		t.Errorf("Expected reason 'Need technical expertise', got '%v'", reason)
	}
}

func TestTransferTool_MinimalArgs(t *testing.T) {
	tool := NewTransferTool("support", "")
	
	ctx := runtime.NewContext(context.Background(), "test-session", "test-trace")
	
	// Minimal arguments (empty object)
	input, _ := json.Marshal(map[string]interface{}{})
	
	result, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	// Verify the result
	var resultStr string
	if err := json.Unmarshal(result, &resultStr); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	
	expected := "Transferring to support agent..."
	if resultStr != expected {
		t.Errorf("Expected result '%s', got '%s'", expected, resultStr)
	}
	
	// Inspect the handoff
	handoffValue, exists := ctx.State().Get("pending_handoff")
	if !exists {
		t.Fatal("Expected pending_handoff to be set in context")
	}
	
	handoff, ok := handoffValue.(*schema.Handoff)
	if !ok {
		t.Fatal("Expected pending_handoff to be *schema.Handoff")
	}
	
	if handoff.Target != "support" {
		t.Errorf("Expected handoff target 'support', got '%s'", handoff.Target)
	}
}

func TestCreateTransferTools(t *testing.T) {
	targets := map[string]string{
		"writer":     "Content writer for articles",
		"researcher": "Research specialist",
		"engineer":   "Technical engineer",
	}
	
	tools := CreateTransferTools(targets)
	
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}
	
	// Verify tool names
	expectedNames := []string{"transfer_to_writer", "transfer_to_researcher", "transfer_to_engineer"}
	actualNames := make([]string, len(tools))
	for i, tool := range tools {
		actualNames[i] = tool.Name()
	}
	
	for _, expected := range expectedNames {
		found := false
		for _, actual := range actualNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool name '%s' not found in %v", expected, actualNames)
		}
	}
}

func TestGetCommonTransferTools(t *testing.T) {
	tools := GetCommonTransferTools()
	
	if len(tools) < 5 {
		t.Errorf("Expected at least 5 common tools, got %d", len(tools))
	}
	
	// Ensure common tools are included
	expectedTools := []string{
		"transfer_to_writer",
		"transfer_to_researcher", 
		"transfer_to_engineer",
		"transfer_to_expert",
		"transfer_to_supervisor",
	}
	
	actualNames := make([]string, len(tools))
	for i, tool := range tools {
		actualNames[i] = tool.Name()
	}
	
	for _, expected := range expectedTools {
		found := false
		for _, actual := range actualNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected common tool '%s' not found in %v", expected, actualNames)
		}
	}
}

func TestExtractHandoffFromToolCall(t *testing.T) {
	// Validate a proper transfer tool invocation
	args := map[string]interface{}{
		"reason":   "Complex technical issue",
		"priority": 9,
		"context": map[string]interface{}{
			"urgency": "high",
		},
	}
	
	argsBytes, _ := json.Marshal(args)
	
	toolCall := schema.ToolCall{
		ID:   "call_123",
		Name: "transfer_to_engineer",
		Args: argsBytes,
	}
	
	handoff, err := ExtractHandoffFromToolCall(toolCall)
	if err != nil {
		t.Fatalf("ExtractHandoffFromToolCall failed: %v", err)
	}
	
	if handoff.Target != "engineer" {
		t.Errorf("Expected target 'engineer', got '%s'", handoff.Target)
	}
	
	if handoff.Priority != 9 {
		t.Errorf("Expected priority 9, got %d", handoff.Priority)
	}
	
	reason, exists := handoff.GetContext("reason")
	if !exists || reason != "Complex technical issue" {
		t.Errorf("Expected reason 'Complex technical issue', got '%v'", reason)
	}
}

func TestIsTransferTool(t *testing.T) {
	testCases := []struct {
		toolName string
		expected bool
	}{
		{"transfer_to_writer", true},
		{"transfer_to_engineer", true},
		{"calculator", false},
		{"transfer_", false},
		{"transfer_to", false},
		{"", false},
	}
	
	for _, tc := range testCases {
		result := IsTransferTool(tc.toolName)
		if result != tc.expected {
			t.Errorf("IsTransferTool('%s') = %v, expected %v", tc.toolName, result, tc.expected)
		}
	}
}

func TestExtractTargetFromToolName(t *testing.T) {
	testCases := []struct {
		toolName string
		expected string
	}{
		{"transfer_to_writer", "writer"},
		{"transfer_to_engineer", "engineer"},
		{"transfer_to_support_specialist", "support_specialist"},
		{"calculator", ""},
		{"transfer_to", ""},
		{"", ""},
	}
	
	for _, tc := range testCases {
		result := ExtractTargetFromToolName(tc.toolName)
		if result != tc.expected {
			t.Errorf("ExtractTargetFromToolName('%s') = '%s', expected '%s'", tc.toolName, result, tc.expected)
		}
	}
}
