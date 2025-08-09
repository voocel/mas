package mas

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkflowContext represents the execution context for a workflow
type WorkflowContext struct {
	ID       string         `json:"id"`
	Data     map[string]any `json:"data"`
	Messages []Message      `json:"messages"`
	mutex    sync.RWMutex
}

// Get safely retrieves a value from context
func (c *WorkflowContext) Get(key string) any {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Data[key]
}

// Set safely sets a value in context
func (c *WorkflowContext) Set(key string, value any) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Data[key] = value
}

// AddMessage adds a message to the context
func (c *WorkflowContext) AddMessage(role, content string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

// WorkflowNode represents a node in the workflow
type WorkflowNode interface {
	// ID returns the unique identifier of the node
	ID() string

	// Execute runs the node with given context
	Execute(ctx context.Context, wfCtx *WorkflowContext) error
}

// WorkflowBuilder provides a fluent API for building workflows
type WorkflowBuilder struct {
	nodes     map[string]WorkflowNode
	edges     map[string][]string
	startNode string
}

// NewWorkflow creates a new workflow builder
func NewWorkflow() *WorkflowBuilder {
	return &WorkflowBuilder{
		nodes: make(map[string]WorkflowNode),
		edges: make(map[string][]string),
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

// Execute runs the workflow
func (b *WorkflowBuilder) Execute(ctx context.Context, initialData map[string]any) (*WorkflowContext, error) {
	wfCtx := &WorkflowContext{
		ID:   generateWorkflowID(),
		Data: initialData,
	}
	if wfCtx.Data == nil {
		wfCtx.Data = make(map[string]any)
	}

	return b.executeFrom(ctx, b.startNode, wfCtx)
}

// executeFrom executes workflow starting from a specific node
func (b *WorkflowBuilder) executeFrom(ctx context.Context, nodeID string, wfCtx *WorkflowContext) (*WorkflowContext, error) {
	visited := make(map[string]bool)
	queue := []string{nodeID}

	for len(queue) > 0 {
		currentNodeID := queue[0]
		queue = queue[1:]

		if visited[currentNodeID] {
			continue // Avoid cycles
		}
		visited[currentNodeID] = true

		// Check context cancellation
		select {
		case <-ctx.Done():
			return wfCtx, ctx.Err()
		default:
		}

		// Get and execute node
		node, exists := b.nodes[currentNodeID]
		if !exists {
			return wfCtx, fmt.Errorf("node %s not found", currentNodeID)
		}

		if err := node.Execute(ctx, wfCtx); err != nil {
			return wfCtx, fmt.Errorf("node %s execution failed: %w", currentNodeID, err)
		}

		// Add next nodes to queue
		if nextNodes, exists := b.edges[currentNodeID]; exists {
			queue = append(queue, nextNodes...)
		}
	}

	return wfCtx, nil
}

// AgentNode wraps an Agent as a workflow node
type AgentNode struct {
	id     string
	agent  Agent
	prompt string
}

// NewAgentNode creates a new agent workflow node
func NewAgentNode(id string, agent Agent) *AgentNode {
	return &AgentNode{
		id:    id,
		agent: agent,
	}
}

// WithPrompt sets a custom prompt template
func (n *AgentNode) WithPrompt(prompt string) *AgentNode {
	n.prompt = prompt
	return n
}

// ID returns the node ID
func (n *AgentNode) ID() string {
	return n.id
}

// Execute runs the agent
func (n *AgentNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	// Get input from context
	input := wfCtx.Get("input")
	if input == nil {
		input = "Continue with the workflow"
	}

	// Use custom prompt if provided
	prompt := fmt.Sprintf("%v", input)
	if n.prompt != "" {
		prompt = fmt.Sprintf(n.prompt, input)
	}

	// Execute agent
	response, err := n.agent.Chat(ctx, prompt)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	// Store response in context
	wfCtx.Set("output", response)
	wfCtx.Set("last_agent", n.id)
	wfCtx.AddMessage("assistant", response)

	return nil
}

// ToolNode wraps a Tool as a workflow node
type ToolNode struct {
	id     string
	tool   Tool
	params map[string]any
}

// NewToolNode creates a new tool workflow node
func NewToolNode(id string, tool Tool) *ToolNode {
	return &ToolNode{
		id:     id,
		tool:   tool,
		params: make(map[string]any),
	}
}

// WithParams sets tool parameters
func (n *ToolNode) WithParams(params map[string]any) *ToolNode {
	n.params = params
	return n
}

// ID returns the node ID
func (n *ToolNode) ID() string {
	return n.id
}

// Execute runs the tool
func (n *ToolNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	// Merge context data with node params
	params := make(map[string]any)
	for k, v := range n.params {
		params[k] = v
	}

	// Allow context to override params
	if ctxParams := wfCtx.Get("tool_params"); ctxParams != nil {
		if ctxParamsMap, ok := ctxParams.(map[string]any); ok {
			for k, v := range ctxParamsMap {
				params[k] = v
			}
		}
	}

	// Execute tool
	result, err := n.tool.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("tool execution failed: %w", err)
	}

	// Store result in context
	wfCtx.Set("tool_result", result)
	wfCtx.Set("last_tool", n.id)

	return nil
}

// ParallelNode executes multiple nodes concurrently
type ParallelNode struct {
	id    string
	nodes []WorkflowNode
}

// NewParallelNode creates a new parallel workflow node
func NewParallelNode(id string, nodes ...WorkflowNode) *ParallelNode {
	return &ParallelNode{
		id:    id,
		nodes: nodes,
	}
}

// ID returns the node ID
func (n *ParallelNode) ID() string {
	return n.id
}

// Execute runs all child nodes concurrently
func (n *ParallelNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	if len(n.nodes) == 0 {
		return nil
	}

	// Create error channel
	errChan := make(chan error, len(n.nodes))

	// Execute all nodes concurrently
	for _, node := range n.nodes {
		go func(n WorkflowNode) {
			errChan <- n.Execute(ctx, wfCtx)
		}(node)
	}

	// Wait for all nodes to complete
	for i := 0; i < len(n.nodes); i++ {
		if err := <-errChan; err != nil {
			return err // Return first error
		}
	}

	return nil
}

// generateWorkflowID generates a unique workflow ID
func generateWorkflowID() string {
	return fmt.Sprintf("wf_%d", time.Now().UnixNano())
}
