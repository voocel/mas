package workflows

import (
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// Workflow defines the workflow interface
type Workflow interface {
	// Name returns the workflow name
	Name() string

	// Description returns the workflow description
	Description() string

	// Execute runs the workflow
	Execute(ctx runtime.Context, input schema.Message) (schema.Message, error)

	// ExecuteStream runs the workflow in streaming mode
	ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error)

	// Validate verifies the workflow configuration
	Validate() error
}

// WorkflowType enumerates supported workflow types
type WorkflowType string

const (
	WorkflowTypeChain WorkflowType = "chain"
	WorkflowTypeGraph WorkflowType = "graph"
	WorkflowTypeMap   WorkflowType = "map"
)

// WorkflowConfig stores workflow configuration
type WorkflowConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        WorkflowType           `json:"type"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BaseWorkflow provides shared workflow functionality
type BaseWorkflow struct {
	config WorkflowConfig
}

// NewBaseWorkflow constructs a base workflow
func NewBaseWorkflow(config WorkflowConfig) *BaseWorkflow {
	return &BaseWorkflow{
		config: config,
	}
}

func (w *BaseWorkflow) Name() string {
	return w.config.Name
}

func (w *BaseWorkflow) Description() string {
	return w.config.Description
}

func (w *BaseWorkflow) Validate() error {
	if w.config.Name == "" {
		return schema.NewValidationError("name", w.config.Name, "workflow name cannot be empty")
	}
	return nil
}

// Node defines the workflow node interface
type Node interface {
	// Name returns the node name
	Name() string

	// Execute runs the node
	Execute(ctx runtime.Context, input schema.Message) (schema.Message, error)

	// ExecuteStream streams the node
	ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error)

	// Condition optionally determines whether the node executes
	Condition(ctx runtime.Context, input schema.Message) bool
}

// NodeConfig describes node configuration
type NodeConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	Timeout     int                    `json:"timeout"` // Seconds
	Retry       int                    `json:"retry"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BaseNode provides a reusable node base
type BaseNode struct {
	config NodeConfig
}

// NewBaseNode constructs a base node
func NewBaseNode(config NodeConfig) *BaseNode {
	return &BaseNode{
		config: config,
	}
}

func (n *BaseNode) Name() string {
	return n.config.Name
}

func (n *BaseNode) Condition(ctx runtime.Context, input schema.Message) bool {
	// Execute by default
	return true
}

// ExecutionResult records the outcome of a node
type ExecutionResult struct {
	Node     string         `json:"node"`
	Input    schema.Message `json:"input"`
	Output   schema.Message `json:"output"`
	Duration int64          `json:"duration"` // Milliseconds
	Error    string         `json:"error,omitempty"`
	Skipped  bool           `json:"skipped,omitempty"`
}

// WorkflowExecution captures workflow execution state
type WorkflowExecution struct {
	WorkflowName string            `json:"workflow_name"`
	Status       ExecutionStatus   `json:"status"`
	Results      []ExecutionResult `json:"results"`
	StartTime    int64             `json:"start_time"`
	EndTime      int64             `json:"end_time"`
	Error        string            `json:"error,omitempty"`
}

// ExecutionStatus captures workflow progress
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// WorkflowBuilder helps assemble workflows
type WorkflowBuilder struct {
	config WorkflowConfig
	nodes  []Node
}

// NewWorkflowBuilder creates a workflow builder
func NewWorkflowBuilder(name, description string) *WorkflowBuilder {
	return &WorkflowBuilder{
		config: WorkflowConfig{
			Name:        name,
			Description: description,
			Type:        WorkflowTypeChain, // Chain workflow by default
			Metadata:    make(map[string]interface{}),
		},
		nodes: make([]Node, 0),
	}
}

// WithType sets the workflow type
func (b *WorkflowBuilder) WithType(workflowType WorkflowType) *WorkflowBuilder {
	b.config.Type = workflowType
	return b
}

// WithMetadata sets metadata
func (b *WorkflowBuilder) WithMetadata(key string, value interface{}) *WorkflowBuilder {
	if b.config.Metadata == nil {
		b.config.Metadata = make(map[string]interface{})
	}
	b.config.Metadata[key] = value
	return b
}

// AddNode appends a node
func (b *WorkflowBuilder) AddNode(node Node) *WorkflowBuilder {
	b.nodes = append(b.nodes, node)
	return b
}

// Build assembles the workflow
func (b *WorkflowBuilder) Build() (Workflow, error) {
	switch b.config.Type {
	case WorkflowTypeChain:
		return NewChainWorkflow(b.config, b.nodes), nil
	case WorkflowTypeGraph:
		// Graph workflows should be produced via GraphBuilder
		return nil, schema.NewValidationError("type", b.config.Type, "use GraphBuilder for graph workflow")
	case WorkflowTypeMap:
		// TODO: implement map workflow
		return nil, schema.NewValidationError("type", b.config.Type, "map workflow not implemented yet")
	default:
		return nil, schema.NewValidationError("type", b.config.Type, "unsupported workflow type")
	}
}

// WorkflowOption customizes workflow configuration
type WorkflowOption func(*WorkflowConfig)

// WithWorkflowMetadata sets workflow metadata
func WithWorkflowMetadata(metadata map[string]interface{}) WorkflowOption {
	return func(config *WorkflowConfig) {
		if config.Metadata == nil {
			config.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			config.Metadata[k] = v
		}
	}
}

// NodeOption customizes node configuration
type NodeOption func(*NodeConfig)

// WithNodeTimeout sets the node timeout
func WithNodeTimeout(timeout int) NodeOption {
	return func(config *NodeConfig) {
		config.Timeout = timeout
	}
}

// WithNodeRetry sets the node retry count
func WithNodeRetry(retry int) NodeOption {
	return func(config *NodeConfig) {
		config.Retry = retry
	}
}

// WithNodeRequired toggles whether the node is required
func WithNodeRequired(required bool) NodeOption {
	return func(config *NodeConfig) {
		config.Required = required
	}
}

// WithNodeMetadata merges node metadata
func WithNodeMetadata(metadata map[string]interface{}) NodeOption {
	return func(config *NodeConfig) {
		if config.Metadata == nil {
			config.Metadata = make(map[string]interface{})
		}
		for k, v := range metadata {
			config.Metadata[k] = v
		}
	}
}

// NewNodeConfig constructs a node configuration
func NewNodeConfig(name, description string, options ...NodeOption) NodeConfig {
	config := NodeConfig{
		Name:        name,
		Description: description,
		Required:    true,
		Timeout:     30, // Default 30 seconds
		Retry:       0,  // Default no retries
		Metadata:    make(map[string]interface{}),
	}

	for _, option := range options {
		option(&config)
	}

	return config
}

// NewWorkflowConfig constructs a workflow configuration
func NewWorkflowConfig(name, description string, workflowType WorkflowType, options ...WorkflowOption) WorkflowConfig {
	config := WorkflowConfig{
		Name:        name,
		Description: description,
		Type:        workflowType,
		Metadata:    make(map[string]interface{}),
	}

	for _, option := range options {
		option(&config)
	}

	return config
}
