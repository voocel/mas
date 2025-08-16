package workflow

import (
	"context"
	"fmt"
	"time"
)

// WorkflowNode represents a node in the workflow
type WorkflowNode interface {
	ID() string
	Execute(ctx context.Context, wfCtx *WorkflowContext) error
}

// WorkflowBuilder provides a fluent API for building workflows
type WorkflowBuilder struct {
	nodes             map[string]WorkflowNode
	edges             map[string][]string
	startNode         string
	checkpointer      Checkpointer
	checkpointConfig  CheckpointConfig
	enableCheckpoints bool
}

// Checkpointer interface placeholder (will be satisfied by mas.Checkpointer)
type Checkpointer interface {
	Save(ctx context.Context, checkpoint interface{}) error
	Load(ctx context.Context, workflowID string) (interface{}, error)
}

// CheckpointConfig placeholder (will be satisfied by mas.CheckpointConfig)
type CheckpointConfig struct {
	AutoSave       bool
	SaveInterval   time.Duration
	MaxCheckpoints int
	Compression    bool
	SaveBeforeNode bool
	SaveAfterNode  bool
}

// NewBuilder creates a new workflow builder
func NewBuilder() *WorkflowBuilder {
	return &WorkflowBuilder{
		nodes:             make(map[string]WorkflowNode),
		edges:             make(map[string][]string),
		checkpointConfig:  DefaultCheckpointConfig(),
		enableCheckpoints: false,
	}
}

// AddNode adds a node to the workflow
func (b *WorkflowBuilder) AddNode(node WorkflowNode) *WorkflowBuilder {
	b.nodes[node.ID()] = node
	return b
}

// AddEdge adds an edge between nodes
func (b *WorkflowBuilder) AddEdge(from, to string) *WorkflowBuilder {
	b.edges[from] = append(b.edges[from], to)
	return b
}

// SetStart sets the starting node
func (b *WorkflowBuilder) SetStart(nodeID string) *WorkflowBuilder {
	b.startNode = nodeID
	return b
}

// WithCheckpointer enables checkpointing with the specified checkpointer
func (b *WorkflowBuilder) WithCheckpointer(checkpointer Checkpointer) *WorkflowBuilder {
	b.checkpointer = checkpointer
	b.enableCheckpoints = true
	return b
}

// AddConditionalRoute adds a simple conditional route
func (b *WorkflowBuilder) AddConditionalRoute(fromNodeID string, condition func(*WorkflowContext) bool, trueTarget, falseTarget string) *WorkflowBuilder {
	// Create a conditional node
	conditionalNodeID := fromNodeID + "_conditional"
	conditionalNode := NewConditionalNode(conditionalNodeID)
	conditionalNode.When(condition, trueTarget)
	conditionalNode.Otherwise(falseTarget)

	// Add the conditional node and connect it
	b.nodes[conditionalNodeID] = conditionalNode
	b.edges[fromNodeID] = []string{conditionalNodeID}

	return b
}

// Execute runs the workflow
func (b *WorkflowBuilder) Execute(ctx context.Context, initialData map[string]any) (*WorkflowContext, error) {
	wfCtx := NewWorkflowContext(generateWorkflowID(), initialData)
	return b.executeFrom(ctx, b.startNode, wfCtx)
}

// ExecuteWithCheckpoint runs the workflow with checkpoint support
func (b *WorkflowBuilder) ExecuteWithCheckpoint(ctx context.Context, initialData map[string]any) (*WorkflowContext, error) {
	if b.checkpointer == nil {
		return nil, fmt.Errorf("no checkpointer configured - use WithCheckpointer() to enable checkpointing")
	}

	b.enableCheckpoints = true
	wfCtx := NewWorkflowContext(generateWorkflowID(), initialData)

	// Save initial checkpoint if configured
	if b.checkpointConfig.AutoSave {
		// Create and save initial checkpoint
		// This would use the actual checkpoint creation logic
	}

	return b.executeFrom(ctx, b.startNode, wfCtx)
}

// ResumeFromCheckpoint resumes workflow execution from the latest checkpoint
func (b *WorkflowBuilder) ResumeFromCheckpoint(ctx context.Context, workflowID string) (*WorkflowContext, error) {
	if b.checkpointer == nil {
		return nil, fmt.Errorf("no checkpointer configured - use WithCheckpointer() to enable checkpointing")
	}

	// Load the latest checkpoint
	_, err := b.checkpointer.Load(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint for workflow %s: %w", workflowID, err)
	}

	// This would contain the actual resume logic
	// For now, return a basic implementation
	return &WorkflowContext{ID: workflowID}, nil
}

// DefaultCheckpointConfig returns a default checkpoint configuration
func DefaultCheckpointConfig() CheckpointConfig {
	return CheckpointConfig{
		AutoSave:       true,
		SaveInterval:   30 * time.Second,
		MaxCheckpoints: 10,
		Compression:    true,
		SaveBeforeNode: false,
		SaveAfterNode:  true,
	}
}

// generateWorkflowID generates a unique workflow ID
func generateWorkflowID() string {
	return fmt.Sprintf("wf_%d", time.Now().UnixNano())
}
