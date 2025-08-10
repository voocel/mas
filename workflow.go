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

// AddConditionalEdge adds a conditional edge with multiple conditions
func (b *WorkflowBuilder) AddConditionalEdge(fromNodeID string, conditions ...Condition) *WorkflowBuilder {
	// Create a conditional node
	conditionalNodeID := fromNodeID + "_conditional"
	conditionalNode := NewConditionalNode(conditionalNodeID)

	for _, condition := range conditions {
		conditionalNode.When(condition.Check, condition.Target)
	}

	// Add the conditional node and connect it
	b.nodes[conditionalNodeID] = conditionalNode
	b.edges[fromNodeID] = []string{conditionalNodeID}

	return b
}

// When creates a condition for use with AddConditionalEdge
func When(check func(*WorkflowContext) bool, target string) Condition {
	return Condition{
		Check:  check,
		Target: target,
	}
}

// AddConditionalRoute adds a simple conditional route (convenience method)
func (b *WorkflowBuilder) AddConditionalRoute(fromNodeID string, condition func(*WorkflowContext) bool, trueTarget, falseTarget string) *WorkflowBuilder {
	return b.AddConditionalEdge(fromNodeID,
		When(condition, trueTarget),
		When(func(*WorkflowContext) bool { return true }, falseTarget), // Default case
	)
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

		// Check if node specified next node (for conditional routing)
		if nextNode := wfCtx.Get("next_node"); nextNode != nil {
			if nextNodeStr, ok := nextNode.(string); ok && nextNodeStr != "" {
				wfCtx.Set("next_node", nil) // Clear for next iteration
				queue = append(queue, nextNodeStr)
				continue
			}
		}

		// Add next nodes to queue based on edges
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
	input := wfCtx.Get("input")
	if input == nil {
		input = "Continue with the workflow"
	}

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

// ConditionFunc represents a condition function for routing
type ConditionFunc func(*WorkflowContext) string

// ConditionalNode makes routing decisions based on context state
type ConditionalNode struct {
	id         string
	conditions []Condition
	defaultTo  string
}

// Condition represents a single condition with its target
type Condition struct {
	Check  func(*WorkflowContext) bool
	Target string
}

// NewConditionalNode creates a new conditional routing node
func NewConditionalNode(id string) *ConditionalNode {
	return &ConditionalNode{
		id:         id,
		conditions: make([]Condition, 0),
	}
}

// When adds a condition with its target node
func (n *ConditionalNode) When(check func(*WorkflowContext) bool, target string) *ConditionalNode {
	n.conditions = append(n.conditions, Condition{
		Check:  check,
		Target: target,
	})
	return n
}

// Otherwise sets the default target when no conditions match
func (n *ConditionalNode) Otherwise(target string) *ConditionalNode {
	n.defaultTo = target
	return n
}

// ID returns the node ID
func (n *ConditionalNode) ID() string {
	return n.id
}

// Execute evaluates conditions and returns the target node
func (n *ConditionalNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	// Find the first matching condition
	for _, condition := range n.conditions {
		if condition.Check(wfCtx) {
			wfCtx.Set("next_node", condition.Target)
			return nil
		}
	}

	// Use default if no conditions match
	if n.defaultTo != "" {
		wfCtx.Set("next_node", n.defaultTo)
		return nil
	}

	return fmt.Errorf("no matching condition and no default route in conditional node %s", n.id)
}

// HumanInput represents input from a human user
type HumanInput struct {
	Value string
	Data  map[string]any
}

// HumanInputProvider handles human input collection
type HumanInputProvider interface {
	// RequestInput requests input from a human with a prompt
	RequestInput(ctx context.Context, prompt string, options ...HumanInputOption) (*HumanInput, error)
}

// HumanInputOption configures human input behavior
type HumanInputOption func(*HumanInputConfig)

// HumanInputConfig contains configuration for human input
type HumanInputConfig struct {
	Timeout   time.Duration
	Validator func(string) error
	Required  bool
}

// WithTimeout sets the timeout for human input
func WithTimeout(timeout time.Duration) HumanInputOption {
	return func(config *HumanInputConfig) {
		config.Timeout = timeout
	}
}

// WithValidator sets a validation function for human input
func WithValidator(validator func(string) error) HumanInputOption {
	return func(config *HumanInputConfig) {
		config.Validator = validator
	}
}

// WithRequired sets whether input is required
func WithRequired(required bool) HumanInputOption {
	return func(config *HumanInputConfig) {
		config.Required = required
	}
}

// DefaultHumanInputConfig returns default configuration
func DefaultHumanInputConfig() HumanInputConfig {
	return HumanInputConfig{
		Timeout:  5 * time.Minute,
		Required: true,
	}
}

// HumanNode represents a node that requires human input
type HumanNode struct {
	id       string
	prompt   string
	provider HumanInputProvider
	options  []HumanInputOption
}

// NewHumanNode creates a new human input node
func NewHumanNode(id, prompt string, provider HumanInputProvider) *HumanNode {
	return &HumanNode{
		id:       id,
		prompt:   prompt,
		provider: provider,
		options:  make([]HumanInputOption, 0),
	}
}

// WithOptions adds configuration options
func (n *HumanNode) WithOptions(options ...HumanInputOption) *HumanNode {
	n.options = append(n.options, options...)
	return n
}

// ID returns the node ID
func (n *HumanNode) ID() string {
	return n.id
}

// Execute requests human input and stores the result
func (n *HumanNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	// Build prompt with context data
	prompt := n.prompt
	if contextData := wfCtx.Get("output"); contextData != nil {
		prompt = fmt.Sprintf("%s\n\nContext: %v", prompt, contextData)
	}

	// Request human input
	input, err := n.provider.RequestInput(ctx, prompt, n.options...)
	if err != nil {
		return fmt.Errorf("human input failed: %w", err)
	}

	// Store input in context
	wfCtx.Set("human_input", input.Value)
	wfCtx.Set("human_data", input.Data)
	wfCtx.AddMessage("human", input.Value)

	return nil
}

// ConsoleInputProvider provides human input via console
type ConsoleInputProvider struct{}

// NewConsoleInputProvider creates a new console input provider
func NewConsoleInputProvider() *ConsoleInputProvider {
	return &ConsoleInputProvider{}
}

// RequestInput requests input from console with timeout
func (p *ConsoleInputProvider) RequestInput(ctx context.Context, prompt string, options ...HumanInputOption) (*HumanInput, error) {
	config := DefaultHumanInputConfig()
	for _, option := range options {
		option(&config)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	inputChan := make(chan *HumanInput, 1)
	errChan := make(chan error, 1)

	// Start input goroutine
	go func() {
		fmt.Printf("\nHuman Input Required:\n%s\n> ", prompt)

		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			errChan <- fmt.Errorf("failed to read input: %w", err)
			return
		}

		if config.Validator != nil {
			if err := config.Validator(input); err != nil {
				errChan <- fmt.Errorf("validation failed: %w", err)
				return
			}
		}

		if config.Required && input == "" {
			errChan <- fmt.Errorf("input is required")
			return
		}

		inputChan <- &HumanInput{
			Value: input,
			Data:  make(map[string]any),
		}
	}()

	select {
	case input := <-inputChan:
		return input, nil
	case err := <-errChan:
		return nil, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("human input timeout after %v", config.Timeout)
	}
}

// generateWorkflowID generates a unique workflow ID
func generateWorkflowID() string {
	return fmt.Sprintf("wf_%d", time.Now().UnixNano())
}
