package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/workflows"
)

type TestModel struct {
	*llm.BaseModel
	response string
}

func NewTestModel(response string) *TestModel {
	info := llm.ModelInfo{
		Name:         "test-model",
		Provider:     "test",
		Version:      "1.0",
		MaxTokens:    1000,
		ContextSize:  4000,
		Capabilities: []string{"chat", "completion"},
	}

	return &TestModel{
		BaseModel: llm.NewBaseModel(info, llm.DefaultGenerationConfig),
		response:  response,
	}
}

func (m *TestModel) Generate(ctx runtime.Context, messages []schema.Message) (schema.Message, error) {
	return schema.Message{
		Role:      schema.RoleAssistant,
		Content:   m.response,
		Timestamp: time.Now(),
	}, nil
}

func (m *TestModel) GenerateStream(ctx runtime.Context, messages []schema.Message) (<-chan schema.StreamEvent, error) {
	eventChan := make(chan schema.StreamEvent, 10)

	go func() {
		defer close(eventChan)
		eventChan <- schema.NewStreamEvent(schema.EventStart, nil)
		eventChan <- schema.NewTokenEvent(m.response, m.response, "")
		eventChan <- schema.NewStreamEvent(schema.EventEnd, schema.Message{
			Role:      schema.RoleAssistant,
			Content:   m.response,
			Timestamp: time.Now(),
		})
	}()

	return eventChan, nil
}

func TestBasicOrchestrator(t *testing.T) {
	orch := NewOrchestrator()
	model := NewTestModel("Hello from test agent")
	testAgent := agent.NewAgent("test", "Test Agent", model)

	// Add agent
	err := orch.AddAgent("test", testAgent)
	if err != nil {
		t.Fatalf("Failed to add agent: %v", err)
	}

	// Verify agent was added
	agents := orch.ListAgents()
	if len(agents) != 1 || agents[0] != "test" {
		t.Errorf("Expected 1 agent named 'test', got %v", agents)
	}

	// Get agent
	retrievedAgent, exists := orch.GetAgent("test")
	if !exists {
		t.Error("Agent should exist")
	}
	if retrievedAgent.ID() != "test" {
		t.Errorf("Expected agent ID 'test', got %s", retrievedAgent.ID())
	}

	// Execute agent
	ctx := runtime.NewContext(context.Background(), "test-session", "test-trace")
	request := ExecuteRequest{
		Input: schema.Message{
			Role:    schema.RoleUser,
			Content: "Hello",
		},
		Target: "test",
		Type:   ExecuteTypeAgent,
	}

	response, err := orch.Execute(ctx, request)
	if err != nil {
		t.Fatalf("Failed to execute agent: %v", err)
	}

	if response.Output.Content != "Hello from test agent" {
		t.Errorf("Expected 'Hello from test agent', got %s", response.Output.Content)
	}

	// Remove agent
	err = orch.RemoveAgent("test")
	if err != nil {
		t.Fatalf("Failed to remove agent: %v", err)
	}

	// Verify agent was removed
	agents = orch.ListAgents()
	if len(agents) != 0 {
		t.Errorf("Expected 0 agents, got %v", agents)
	}
}

func TestOrchestratorAutoExecution(t *testing.T) {
	orch := NewOrchestrator()

	model := NewTestModel("Agent response")
	testAgent := agent.NewAgent("auto-test", "Auto Test Agent", model)
	orch.AddAgent("auto-test", testAgent)

	// Add workflow
	step := workflows.NewFunctionNode(
		workflows.NewNodeConfig("test-step", "Test step"),
		func(ctx runtime.Context, input schema.Message) (schema.Message, error) {
			return schema.Message{
				Role:    schema.RoleAssistant,
				Content: "Workflow response",
			}, nil
		},
	)

	workflow := workflows.NewChainBuilder("auto-workflow", "Auto test workflow").
		Then(step).
		Build()

	orch.AddWorkflow("auto-workflow", workflow)

	ctx := runtime.NewContext(context.Background(), "auto-session", "auto-trace")

	// Test auto-selection of agent
	agentRequest := ExecuteRequest{
		Input: schema.Message{
			Role:    schema.RoleUser,
			Content: "Test agent",
		},
		Target: "auto-test",
		Type:   ExecuteTypeAuto,
	}

	response, err := orch.Execute(ctx, agentRequest)
	if err != nil {
		t.Fatalf("Failed to auto-execute agent: %v", err)
	}

	if response.Output.Content != "Agent response" {
		t.Errorf("Expected 'Agent response', got %s", response.Output.Content)
	}

	// Test auto-selection of workflow
	workflowRequest := ExecuteRequest{
		Input: schema.Message{
			Role:    schema.RoleUser,
			Content: "Test workflow",
		},
		Target: "auto-workflow",
		Type:   ExecuteTypeAuto,
	}

	response, err = orch.Execute(ctx, workflowRequest)
	if err != nil {
		t.Fatalf("Failed to auto-execute workflow: %v", err)
	}

	if response.Output.Content != "Workflow response" {
		t.Errorf("Expected 'Workflow response', got %s", response.Output.Content)
	}
}

func TestOrchestratorErrors(t *testing.T) {
	orch := NewOrchestrator()

	ctx := runtime.NewContext(context.Background(), "error-session", "error-trace")

	// Test non-existent agent
	request := ExecuteRequest{
		Input: schema.Message{
			Role:    schema.RoleUser,
			Content: "Test",
		},
		Target: "nonexistent",
		Type:   ExecuteTypeAgent,
	}

	_, err := orch.Execute(ctx, request)
	if err == nil {
		t.Error("Expected error for nonexistent agent")
	}

	// Test non-existent workflow
	request.Type = ExecuteTypeWorkflow
	_, err = orch.Execute(ctx, request)
	if err == nil {
		t.Error("Expected error for nonexistent workflow")
	}

	// Test invalid execution type
	request.Type = "invalid"
	_, err = orch.Execute(ctx, request)
	if err == nil {
		t.Error("Expected error for invalid execute type")
	}

	// Test adding an empty agent
	err = orch.AddAgent("", nil)
	if err == nil {
		t.Error("Expected error for empty agent name")
	}

	err = orch.AddAgent("test", nil)
	if err == nil {
		t.Error("Expected error for nil agent")
	}

	// Test adding a duplicate agent
	model := NewTestModel("Test")
	testAgent := agent.NewAgent("duplicate", "Duplicate Agent", model)

	err = orch.AddAgent("duplicate", testAgent)
	if err != nil {
		t.Fatalf("Failed to add first agent: %v", err)
	}

	err = orch.AddAgent("duplicate", testAgent)
	if err == nil {
		t.Error("Expected error for duplicate agent")
	}
}

func TestChainWorkflow(t *testing.T) {
	// Create test steps
	step1 := workflows.NewFunctionNode(
		workflows.NewNodeConfig("step1", "First step"),
		func(ctx runtime.Context, input schema.Message) (schema.Message, error) {
			return schema.Message{
				Role:    schema.RoleSystem,
				Content: "Step1: " + input.Content,
			}, nil
		},
	)

	step2 := workflows.NewFunctionNode(
		workflows.NewNodeConfig("step2", "Second step"),
		func(ctx runtime.Context, input schema.Message) (schema.Message, error) {
			return schema.Message{
				Role:    schema.RoleAssistant,
				Content: "Step2: " + input.Content,
			}, nil
		},
	)

	// Build the workflow
	workflow := workflows.NewChainBuilder("test-chain", "Test chain workflow").
		Then(step1).
		Then(step2).
		Build()

	// Validate workflow properties
	if workflow.Name() != "test-chain" {
		t.Errorf("Expected workflow name 'test-chain', got %s", workflow.Name())
	}

	if workflow.Description() != "Test chain workflow" {
		t.Errorf("Expected description 'Test chain workflow', got %s", workflow.Description())
	}

	// Validate the workflow
	err := workflow.Validate()
	if err != nil {
		t.Fatalf("Workflow validation failed: %v", err)
	}

	// Execute the workflow
	ctx := runtime.NewContext(context.Background(), "workflow-session", "workflow-trace")
	input := schema.Message{
		Role:    schema.RoleUser,
		Content: "Original",
	}

	output, err := workflow.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Workflow execution failed: %v", err)
	}

	expected := "Step2: Step1: Original"
	if output.Content != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output.Content)
	}

	// Validate the number of steps
	steps := workflow.GetNodes()
	if len(steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(steps))
	}
}

func TestWorkflowBuilder(t *testing.T) {
	builder := workflows.NewWorkflowBuilder("builder-test", "Builder test workflow")

	step := workflows.NewFunctionNode(
		workflows.NewNodeConfig("builder-step", "Builder step"),
		func(ctx runtime.Context, input schema.Message) (schema.Message, error) {
			return input, nil
		},
	)

	workflow, err := builder.
		WithType(workflows.WorkflowTypeChain).
		WithMetadata("version", "1.0").
		AddNode(step).
		Build()

	if err != nil {
		t.Fatalf("Failed to build workflow: %v", err)
	}

	if workflow.Name() != "builder-test" {
		t.Errorf("Expected workflow name 'builder-test', got %s", workflow.Name())
	}

	// Test unsupported workflow type
	_, err = builder.
		WithType(workflows.WorkflowTypeGraph).
		Build()

	if err == nil {
		t.Error("Expected error for unsupported workflow type")
	}
}
