package workflows

import (
	"fmt"
	"sync"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// ChainWorkflow implements a linear workflow
type ChainWorkflow struct {
	*BaseWorkflow
	nodes []Node
}

// NewChainWorkflow constructs a linear workflow
func NewChainWorkflow(config WorkflowConfig, nodes []Node) *ChainWorkflow {
	return &ChainWorkflow{
		BaseWorkflow: NewBaseWorkflow(config),
		nodes:        nodes,
	}
}

// Execute runs the linear workflow
func (w *ChainWorkflow) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	if err := w.Validate(); err != nil {
		return schema.Message{}, err
	}

	if len(w.nodes) == 0 {
		return input, nil // No nodes present, return the input
	}

	currentInput := input

	// Execute each node sequentially
	for i, node := range w.nodes {
		if !node.Condition(ctx, currentInput) {
			// Skip the node when the condition is not met
			continue
		}

		// Execute the node
		output, err := node.Execute(ctx, currentInput)
		if err != nil {
			return schema.Message{}, schema.NewWorkflowError(w.Name(), "execute_node", err)
		}

		// Use the output as the input for the next step
		currentInput = output

		// Record the execution trace
		ctx.State().Set(w.getNodeKey(i), ExecutionResult{
			Node:   node.Name(),
			Input:  input,
			Output: output,
		})
	}

	return currentInput, nil
}

// ExecuteStream runs the workflow in streaming mode
func (w *ChainWorkflow) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	if err := w.Validate(); err != nil {
		return nil, err
	}

	eventChan := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		// Emit the start event
		eventChan <- schema.NewStreamEvent(schema.EventStart, nil)

		if len(w.nodes) == 0 {
			// No nodes present, return the input
			eventChan <- schema.NewStreamEvent(schema.EventEnd, input)
			return
		}

		currentInput := input

		// Execute each node sequentially
		for i, node := range w.nodes {
			// Check the execution condition
			if !node.Condition(ctx, currentInput) {
				// Emit a skip event
				eventChan <- schema.NewStreamEvent(schema.EventStepSkipped, map[string]interface{}{
					"node":   node.Name(),
					"reason": "condition not met",
				})
				continue
			}

			// Emit the step-start event
			eventChan <- schema.NewStreamEvent(schema.EventStepStart, map[string]interface{}{
				"node":  node.Name(),
				"index": i,
			})

			// Execute the node in streaming mode
			nodeEventChan, err := node.ExecuteStream(ctx, currentInput)
			if err != nil {
				eventChan <- schema.NewErrorEvent(err, node.Name())
				return
			}

			var nodeOutput schema.Message

			// Relay node events
			for nodeEvent := range nodeEventChan {
				switch nodeEvent.Type {
				case schema.EventEnd:
					// When the node completes, capture its output
					if msg, ok := nodeEvent.Data.(schema.Message); ok {
						nodeOutput = msg
					}

					// Emit the step-end event
					eventChan <- schema.NewStreamEvent(schema.EventStepEnd, map[string]interface{}{
						"node":   node.Name(),
						"index":  i,
						"output": nodeOutput,
					})
				case schema.EventError:
					eventChan <- nodeEvent
					return
				default:
					// Relay other events
					eventChan <- nodeEvent
				}
			}

			// Use the output as the input for the next step
			currentInput = nodeOutput

			// Record the execution trace
			ctx.State().Set(w.getNodeKey(i), ExecutionResult{
				Node:   node.Name(),
				Input:  input,
				Output: nodeOutput,
			})
		}

		// Emit the workflow completion event
		eventChan <- schema.NewStreamEvent(schema.EventEnd, currentInput)
	}()

	return eventChan, nil
}

// Validate checks the workflow definition
func (w *ChainWorkflow) Validate() error {
	if err := w.BaseWorkflow.Validate(); err != nil {
		return err
	}

	// Ensure node names are unique
	nodeNames := make(map[string]bool)
	for _, node := range w.nodes {
		if nodeNames[node.Name()] {
			return schema.NewValidationError("nodes", node.Name(), "duplicate node name")
		}
		nodeNames[node.Name()] = true
	}

	return nil
}

// getNodeKey generates the state key for a node
func (w *ChainWorkflow) getNodeKey(index int) string {
	return "node_" + string(rune(index))
}

// GetNodes returns the workflow nodes
func (w *ChainWorkflow) GetNodes() []Node {
	return w.nodes
}

// AddNode appends a node to the workflow
func (w *ChainWorkflow) AddNode(node Node) {
	w.nodes = append(w.nodes, node)
}

// InsertNode inserts a node at the specified position
func (w *ChainWorkflow) InsertNode(index int, node Node) error {
	if index < 0 || index > len(w.nodes) {
		return schema.NewValidationError("index", index, "invalid node index")
	}

	// Expand the slice
	w.nodes = append(w.nodes, nil)

	// Shift existing elements
	copy(w.nodes[index+1:], w.nodes[index:])

	// Insert the new node
	w.nodes[index] = node

	return nil
}

// RemoveNode removes a node
func (w *ChainWorkflow) RemoveNode(index int) error {
	if index < 0 || index >= len(w.nodes) {
		return schema.NewValidationError("index", index, "invalid node index")
	}

	// Remove the node from the slice
	w.nodes = append(w.nodes[:index], w.nodes[index+1:]...)

	return nil
}

// ChainBuilder builds chain workflows
type ChainBuilder struct {
	config WorkflowConfig
	nodes  []Node
}

// NewChainBuilder creates a chain workflow builder
func NewChainBuilder(name, description string) *ChainBuilder {
	return &ChainBuilder{
		config: WorkflowConfig{
			Name:        name,
			Description: description,
			Type:        WorkflowTypeChain,
			Metadata:    make(map[string]interface{}),
		},
		nodes: make([]Node, 0),
	}
}

// WithMetadata sets metadata
func (b *ChainBuilder) WithMetadata(key string, value interface{}) *ChainBuilder {
	if b.config.Metadata == nil {
		b.config.Metadata = make(map[string]interface{})
	}
	b.config.Metadata[key] = value
	return b
}

// Then appends a node (fluent API)
func (b *ChainBuilder) Then(node Node) *ChainBuilder {
	b.nodes = append(b.nodes, node)
	return b
}

// ThenAgent adds an agent directly as a node (convenience method)
func (b *ChainBuilder) ThenAgent(ag agent.Agent) *ChainBuilder {
	node := NewAgentNode(NodeConfig{
		Name:        ag.Name(),
		Description: fmt.Sprintf("Agent node for %s", ag.Name()),
	}, ag.Name(), func(agentName string) (interface{}, bool) {
		if ag.Name() == agentName {
			return ag, true
		}
		return nil, false
	})
	return b.Then(node)
}

// ThenParallel executes multiple agents in parallel
func (b *ChainBuilder) ThenParallel(agents ...agent.Agent) *ChainBuilder {
	if len(agents) == 0 {
		return b
	}

	// Create parallel node name
	agentNames := make([]string, len(agents))
	for i, ag := range agents {
		agentNames[i] = ag.Name()
	}
	nodeName := fmt.Sprintf("parallel_%v", agentNames)

	node := NewParallelNode(NodeConfig{
		Name:        nodeName,
		Description: fmt.Sprintf("Parallel execution of agents: %v", agentNames),
	}, agents)

	return b.Then(node)
}

// ThenConditional executes conditional branching
func (b *ChainBuilder) ThenConditional(conditionFunc func(ctx runtime.Context, input schema.Message) string, branches map[string][]agent.Agent) *ChainBuilder {
	if len(branches) == 0 {
		return b
	}

	// Create conditional branch node name
	branchNames := make([]string, 0, len(branches))
	for branchName := range branches {
		branchNames = append(branchNames, branchName)
	}
	nodeName := fmt.Sprintf("conditional_%v", branchNames)

	node := NewConditionalNode(NodeConfig{
		Name:        nodeName,
		Description: fmt.Sprintf("Conditional execution with branches: %v", branchNames),
	}, conditionFunc, branches)

	return b.Then(node)
}

// Build assembles the chain workflow
func (b *ChainBuilder) Build() *ChainWorkflow {
	return NewChainWorkflow(b.config, b.nodes)
}

// AgentNode adapts an agent as a workflow node
type AgentNode struct {
	*BaseNode
	agentName string
	getAgent  func(name string) (interface{}, bool) // Function used to resolve the agent
}

// NewAgentNode constructs an agent node
func NewAgentNode(config NodeConfig, agentName string, getAgent func(name string) (interface{}, bool)) *AgentNode {
	return &AgentNode{
		BaseNode:  NewBaseNode(config),
		agentName: agentName,
		getAgent:  getAgent,
	}
}

// Execute runs the agent node
func (n *AgentNode) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	agent, exists := n.getAgent(n.agentName)
	if !exists {
		return schema.Message{}, schema.NewAgentError(n.agentName, "execute", schema.ErrAgentNotFound)
	}

	if ag, ok := agent.(interface {
		Execute(ctx runtime.Context, input schema.Message) (schema.Message, error)
	}); ok {
		return ag.Execute(ctx, input)
	}

	return schema.Message{}, schema.NewAgentError(n.agentName, "execute", schema.ErrAgentNotSupported)
}

// ExecuteStream streams the agent node
func (n *AgentNode) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	agent, exists := n.getAgent(n.agentName)
	if !exists {
		return nil, schema.NewAgentError(n.agentName, "execute_stream", schema.ErrAgentNotFound)
	}

	if ag, ok := agent.(interface {
		ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error)
	}); ok {
		return ag.ExecuteStream(ctx, input)
	}

	return nil, schema.NewAgentError(n.agentName, "execute_stream", schema.ErrAgentNotSupported)
}

// FunctionNode adapts a function as a workflow node
type FunctionNode struct {
	*BaseNode
	fn       func(ctx runtime.Context, input schema.Message) (schema.Message, error)
	streamFn func(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error)
}

// NewFunctionNode constructs a function node
func NewFunctionNode(config NodeConfig, fn func(ctx runtime.Context, input schema.Message) (schema.Message, error)) *FunctionNode {
	return &FunctionNode{
		BaseNode: NewBaseNode(config),
		fn:       fn,
	}
}

// WithStreamFunction registers the streaming function
func (n *FunctionNode) WithStreamFunction(streamFn func(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error)) *FunctionNode {
	n.streamFn = streamFn
	return n
}

// Execute runs the function node
func (n *FunctionNode) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	if n.fn == nil {
		return schema.Message{}, schema.NewValidationError("function", n.fn, "node function cannot be nil")
	}

	return n.fn(ctx, input)
}

// ExecuteStream streams the function node
func (n *FunctionNode) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	if n.streamFn != nil {
		return n.streamFn(ctx, input)
	}

	// If no stream function is provided, approximate it with the regular function
	eventChan := make(chan schema.StreamEvent, 10)

	go func() {
		defer close(eventChan)

		eventChan <- schema.NewStreamEvent(schema.EventStart, nil)

		output, err := n.Execute(ctx, input)
		if err != nil {
			eventChan <- schema.NewErrorEvent(err, n.Name())
			return
		}

		eventChan <- schema.NewStreamEvent(schema.EventEnd, output)
	}()

	return eventChan, nil
}

// WithStateKey adds state key support to existing ChainWorkflow
func (w *ChainWorkflow) WithStateKey(key string) *ChainWorkflow {
	if w.BaseWorkflow.config.Metadata == nil {
		w.BaseWorkflow.config.Metadata = make(map[string]interface{})
	}
	w.BaseWorkflow.config.Metadata["state_key"] = key
	return w
}

// ParallelNode executes multiple agents in parallel
type ParallelNode struct {
	*BaseNode
	agents []agent.Agent
}

// NewParallelNode creates a parallel node
func NewParallelNode(config NodeConfig, agents []agent.Agent) *ParallelNode {
	return &ParallelNode{
		BaseNode: NewBaseNode(config),
		agents:   agents,
	}
}

func (p *ParallelNode) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	if len(p.agents) == 0 {
		return input, nil
	}

	// Execute all agents in parallel
	var wg sync.WaitGroup
	results := make([]schema.Message, len(p.agents))
	errors := make([]error, len(p.agents))

	for i, ag := range p.agents {
		wg.Add(1)
		go func(index int, agent agent.Agent) {
			defer wg.Done()

			// Execute agent
			result, err := agent.Execute(ctx, input)
			results[index] = result
			errors[index] = err
		}(i, ag)
	}

	// Wait for all agents to complete
	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return schema.Message{}, fmt.Errorf("parallel agent %s failed: %v", p.agents[i].Name(), err)
		}
	}

	// Aggregate results
	aggregatedContent := ""
	for i, result := range results {
		if i > 0 {
			aggregatedContent += "\n\n"
		}
		aggregatedContent += fmt.Sprintf("=== %s ===\n%s", p.agents[i].Name(), result.Content)
	}

	return schema.Message{
		Role:    schema.RoleAssistant,
		Content: aggregatedContent,
	}, nil
}

func (p *ParallelNode) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	eventChan := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		// Execute parallel node
		result, err := p.Execute(ctx, input)
		if err != nil {
			eventChan <- schema.NewStreamEvent(schema.EventError, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		eventChan <- schema.NewStreamEvent(schema.EventStepEnd, result)
	}()

	return eventChan, nil
}

func (p *ParallelNode) Condition(ctx runtime.Context, input schema.Message) bool {
	return true // Parallel node always executes by default
}

// ConditionalNode conditional branch node
type ConditionalNode struct {
	*BaseNode
	conditionFunc func(ctx runtime.Context, input schema.Message) string
	branches      map[string][]agent.Agent
}

// NewConditionalNode creates a conditional branch node
func NewConditionalNode(config NodeConfig, conditionFunc func(ctx runtime.Context, input schema.Message) string, branches map[string][]agent.Agent) *ConditionalNode {
	return &ConditionalNode{
		BaseNode:      NewBaseNode(config),
		conditionFunc: conditionFunc,
		branches:      branches,
	}
}

func (c *ConditionalNode) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	// Execute condition function to get branch name
	branchName := c.conditionFunc(ctx, input)
	agents, exists := c.branches[branchName]
	if !exists {
		return schema.Message{}, fmt.Errorf("branch '%s' not found in conditional node", branchName)
	}

	if len(agents) == 0 {
		return input, nil
	}

	// Execute agents in branch sequentially
	currentInput := input
	for _, ag := range agents {
		result, err := ag.Execute(ctx, currentInput)
		if err != nil {
			return schema.Message{}, fmt.Errorf("agent %s in branch '%s' failed: %v", ag.Name(), branchName, err)
		}
		currentInput = result
	}

	return currentInput, nil
}

func (c *ConditionalNode) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	eventChan := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		// Execute conditional branch node
		result, err := c.Execute(ctx, input)
		if err != nil {
			eventChan <- schema.NewStreamEvent(schema.EventError, map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		eventChan <- schema.NewStreamEvent(schema.EventStepEnd, result)
	}()

	return eventChan, nil
}

func (c *ConditionalNode) Condition(ctx runtime.Context, input schema.Message) bool {
	return true // Conditional branch node always executes by default
}
