package workflows

import (
	"fmt"

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

// ThenAgent Directly add the agent as a node
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

// WithStateKey add state key
func (w *ChainWorkflow) WithStateKey(key string) *ChainWorkflow {
	if w.BaseWorkflow.config.Metadata == nil {
		w.BaseWorkflow.config.Metadata = make(map[string]interface{})
	}
	w.BaseWorkflow.config.Metadata["state_key"] = key
	return w
}
