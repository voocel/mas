package workflows

import (
	"fmt"
	"sync"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
)

// GraphWorkflow implements a graph-based workflow
type GraphWorkflow struct {
	*BaseWorkflow
	nodes     map[string]Node
	edges     map[string][]string // from -> []to
	entryNode string
	exitNodes []string
}

// NewGraphWorkflow constructs a graph workflow
func NewGraphWorkflow(config WorkflowConfig) *GraphWorkflow {
	return &GraphWorkflow{
		BaseWorkflow: NewBaseWorkflow(config),
		nodes:        make(map[string]Node),
		edges:        make(map[string][]string),
		exitNodes:    make([]string, 0),
	}
}

// AddNode registers a node
func (g *GraphWorkflow) AddNode(name string, node Node) error {
	if name == "" {
		return schema.NewValidationError("name", name, "node name cannot be empty")
	}
	if node == nil {
		return schema.NewValidationError("node", node, "node cannot be nil")
	}
	if _, exists := g.nodes[name]; exists {
		return schema.NewValidationError("name", name, "node already exists")
	}

	g.nodes[name] = node
	return nil
}

// AddEdge connects two nodes
func (g *GraphWorkflow) AddEdge(from, to string) error {
	if from == "" || to == "" {
		return schema.NewValidationError("edge", fmt.Sprintf("%s->%s", from, to), "edge nodes cannot be empty")
	}
	if _, exists := g.nodes[from]; !exists {
		return schema.NewValidationError("from", from, "from node does not exist")
	}
	if _, exists := g.nodes[to]; !exists {
		return schema.NewValidationError("to", to, "to node does not exist")
	}

	if g.edges[from] == nil {
		g.edges[from] = make([]string, 0)
	}
	g.edges[from] = append(g.edges[from], to)
	return nil
}

// SetEntryNode defines the entry node
func (g *GraphWorkflow) SetEntryNode(name string) error {
	if _, exists := g.nodes[name]; !exists {
		return schema.NewValidationError("entry", name, "entry node does not exist")
	}
	g.entryNode = name
	return nil
}

// AddExitNode registers an exit node
func (g *GraphWorkflow) AddExitNode(name string) error {
	if _, exists := g.nodes[name]; !exists {
		return schema.NewValidationError("exit", name, "exit node does not exist")
	}
	g.exitNodes = append(g.exitNodes, name)
	return nil
}

// Execute runs the graph workflow
func (g *GraphWorkflow) Execute(ctx runtime.Context, input schema.Message) (schema.Message, error) {
	if err := g.Validate(); err != nil {
		return schema.Message{}, err
	}

	// Track execution state
	executed := make(map[string]bool)
	results := make(map[string]schema.Message)

	// Start execution from the entry node
	return g.executeNode(ctx, g.entryNode, input, executed, results)
}

// executeNode runs a single node
func (g *GraphWorkflow) executeNode(ctx runtime.Context, nodeName string, input schema.Message, executed map[string]bool, results map[string]schema.Message) (schema.Message, error) {
	if executed[nodeName] {
		return results[nodeName], nil
	}

	node, exists := g.nodes[nodeName]
	if !exists {
		return schema.Message{}, schema.NewValidationError("node", nodeName, "node not found")
	}

	// Evaluate the execution condition
	if !node.Condition(ctx, input) {
		// Skip the node when its condition fails
		executed[nodeName] = true
		results[nodeName] = input
		return input, nil
	}

	// Execute the node
	output, err := node.Execute(ctx, input)
	if err != nil {
		return schema.Message{}, schema.NewWorkflowError(g.Name(), "execute_node", err)
	}

	// Record the result
	executed[nodeName] = true
	results[nodeName] = output

	// Record the execution trace
	ctx.State().Set(g.getNodeKey(nodeName), ExecutionResult{
		Node:   node.Name(),
		Input:  input,
		Output: output,
	})

	// Return immediately if this is an exit node
	if g.isExitNode(nodeName) {
		return output, nil
	}

	// Execute downstream nodes
	nextNodes := g.edges[nodeName]
	if len(nextNodes) == 0 {
		// No downstream nodes, return the current result
		return output, nil
	}

	if len(nextNodes) == 1 {
		// One downstream node, continue sequentially
		return g.executeNode(ctx, nextNodes[0], output, executed, results)
	}

	// Multiple downstream nodes, run in parallel
	return g.executeParallel(ctx, nextNodes, output, executed, results)
}

// executeParallel runs multiple nodes concurrently
func (g *GraphWorkflow) executeParallel(ctx runtime.Context, nodeNames []string, input schema.Message, executed map[string]bool, results map[string]schema.Message) (schema.Message, error) {
	var wg sync.WaitGroup
	var mutex sync.Mutex
	var firstError error
	nodeResults := make(map[string]schema.Message)

	// Execute all downstream nodes concurrently
	for _, nodeName := range nodeNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			result, err := g.executeNode(ctx, name, input, executed, results)

			mutex.Lock()
			defer mutex.Unlock()

			if err != nil && firstError == nil {
				firstError = err
				return
			}

			nodeResults[name] = result
		}(nodeName)
	}

	wg.Wait()

	if firstError != nil {
		return schema.Message{}, firstError
	}

	// Merge results (simple strategy: return the first result)
	for _, result := range nodeResults {
		return result, nil
	}

	return input, nil
}

// ExecuteStream runs the graph workflow in streaming mode
func (g *GraphWorkflow) ExecuteStream(ctx runtime.Context, input schema.Message) (<-chan schema.StreamEvent, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}

	eventChan := make(chan schema.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		// Emit the start event
		eventChan <- schema.NewStreamEvent(schema.EventStart, nil)

		// Run the graph workflow
		result, err := g.Execute(ctx, input)
		if err != nil {
			eventChan <- schema.NewErrorEvent(err, g.Name())
			return
		}

		// Emit the completion event
		eventChan <- schema.NewStreamEvent(schema.EventEnd, result)
	}()

	return eventChan, nil
}

// Validate checks the graph workflow
func (g *GraphWorkflow) Validate() error {
	if err := g.BaseWorkflow.Validate(); err != nil {
		return err
	}

	// Ensure at least one node exists
	if len(g.nodes) == 0 {
		return schema.NewValidationError("nodes", g.nodes, "graph must have at least one node")
	}

	// Ensure an entry node is defined
	if g.entryNode == "" {
		return schema.NewValidationError("entry", g.entryNode, "graph must have an entry node")
	}

	// Ensure the graph has no cycles (basic check)
	if g.hasCycle() {
		return schema.NewValidationError("cycle", g.edges, "graph contains cycles")
	}

	return nil
}

// hasCycle performs a simple DFS cycle detection
func (g *GraphWorkflow) hasCycle() bool {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for node := range g.nodes {
		if !visited[node] {
			if g.dfsHasCycle(node, visited, recStack) {
				return true
			}
		}
	}

	return false
}

// dfsHasCycle performs depth-first cycle detection
func (g *GraphWorkflow) dfsHasCycle(node string, visited, recStack map[string]bool) bool {
	visited[node] = true
	recStack[node] = true

	for _, neighbor := range g.edges[node] {
		if !visited[neighbor] {
			if g.dfsHasCycle(neighbor, visited, recStack) {
				return true
			}
		} else if recStack[neighbor] {
			return true
		}
	}

	recStack[node] = false
	return false
}

// isExitNode checks whether a node is an exit node
func (g *GraphWorkflow) isExitNode(nodeName string) bool {
	for _, exitNode := range g.exitNodes {
		if exitNode == nodeName {
			return true
		}
	}
	return false
}

// getNodeKey builds the state key for a node
func (g *GraphWorkflow) getNodeKey(nodeName string) string {
	return "node_" + nodeName
}

// GraphBuilder helps assemble graph workflows
type GraphBuilder struct {
	config WorkflowConfig
	graph  *GraphWorkflow
}

// NewGraphBuilder creates a graph workflow builder
func NewGraphBuilder(name, description string) *GraphBuilder {
	config := WorkflowConfig{
		Name:        name,
		Description: description,
		Type:        WorkflowTypeGraph,
		Metadata:    make(map[string]interface{}),
	}

	return &GraphBuilder{
		config: config,
		graph:  NewGraphWorkflow(config),
	}
}

// AddNode registers a node
func (b *GraphBuilder) AddNode(name string, node Node) *GraphBuilder {
	b.graph.AddNode(name, node)
	return b
}

// AddEdge connects two nodes
func (b *GraphBuilder) AddEdge(from, to string) *GraphBuilder {
	b.graph.AddEdge(from, to)
	return b
}

// SetEntry defines the entry node
func (b *GraphBuilder) SetEntry(name string) *GraphBuilder {
	b.graph.SetEntryNode(name)
	return b
}

func (b *GraphBuilder) AddExit(name string) *GraphBuilder {
	b.graph.AddExitNode(name)
	return b
}

func (b *GraphBuilder) WithMetadata(key string, value interface{}) *GraphBuilder {
	if b.config.Metadata == nil {
		b.config.Metadata = make(map[string]interface{})
	}
	b.config.Metadata[key] = value
	return b
}

// Build assembles the graph workflow
func (b *GraphBuilder) Build() *GraphWorkflow {
	return b.graph
}
