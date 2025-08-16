package workflow

import (
	"context"
	"fmt"
	"time"
)

// Agent interface placeholder (will be satisfied by mas.Agent)
type Agent interface {
	Chat(ctx context.Context, message string) (string, error)
	Name() string
}

// Tool interface placeholder (will be satisfied by mas.Tool)
type Tool interface {
	Name() string
	Execute(ctx context.Context, params map[string]any) (any, error)
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

// HumanInputProvider handles human input collection
type HumanInputProvider interface {
	RequestInput(ctx context.Context, prompt string, options ...HumanInputOption) (*HumanInput, error)
}

// HumanInput represents input from a human user
type HumanInput struct {
	Value string
	Data  map[string]any
}

// HumanInputOption configures human input behavior
type HumanInputOption func(*HumanInputConfig)

// HumanInputConfig contains configuration for human input
type HumanInputConfig struct {
	Timeout   time.Duration
	Validator func(string) error
	Required  bool
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

// Human input option helpers
func WithTimeout(timeout time.Duration) HumanInputOption {
	return func(config *HumanInputConfig) {
		config.Timeout = timeout
	}
}

func WithValidator(validator func(string) error) HumanInputOption {
	return func(config *HumanInputConfig) {
		config.Validator = validator
	}
}

func WithRequired(required bool) HumanInputOption {
	return func(config *HumanInputConfig) {
		config.Required = required
	}
}