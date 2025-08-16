package mas

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type workflowBuilder struct {
	nodes             map[string]WorkflowNode
	edges             map[string][]string
	startNode         string
	conditionalRoutes map[string]*conditionalRoute
	checkpointer      Checkpointer
	mu                sync.RWMutex
}

type conditionalRoute struct {
	condition   func(*WorkflowContext) bool
	trueTarget  string
	falseTarget string
}

type agentNode struct {
	id     string
	agent  Agent
	prompt string
}

type toolNode struct {
	id     string
	tool   Tool
	params map[string]any
}

type parallelNode struct {
	id    string
	nodes []WorkflowNode
}

type conditionalNode struct {
	id         string
	conditions []Condition
	defaultTo  string
}

type humanNode struct {
	id       string
	prompt   string
	provider HumanInputProvider
	options  []HumanInputOption
}

// NewWorkflow Create Workflow Builder
func NewWorkflow() WorkflowBuilder {
	return &workflowBuilder{
		nodes:             make(map[string]WorkflowNode),
		edges:             make(map[string][]string),
		conditionalRoutes: make(map[string]*conditionalRoute),
	}
}

// NewWorkflowContext Create Workflow Context
func NewWorkflowContext(id string, initialData map[string]any) *WorkflowContext {
	data := make(map[string]any)
	if initialData != nil {
		for k, v := range initialData {
			data[k] = v
		}
	}

	return &WorkflowContext{
		ID:       id,
		Data:     data,
		Messages: make([]Message, 0),
	}
}

func NewAgentNode(id string, agent Agent) WorkflowNode {
	return &agentNode{
		id:    id,
		agent: agent,
	}
}

func NewToolNode(id string, tool Tool) WorkflowNode {
	return &toolNode{
		id:     id,
		tool:   tool,
		params: make(map[string]any),
	}
}

func NewParallelNode(id string, nodes ...WorkflowNode) WorkflowNode {
	return &parallelNode{
		id:    id,
		nodes: nodes,
	}
}

func NewConditionalNode(id string) WorkflowNode {
	return &conditionalNode{
		id:         id,
		conditions: make([]Condition, 0),
	}
}

func NewHumanNode(id, prompt string, provider HumanInputProvider) WorkflowNode {
	return &humanNode{
		id:       id,
		prompt:   prompt,
		provider: provider,
		options:  make([]HumanInputOption, 0),
	}
}

// Console input provider
type consoleInputProvider struct{}

func NewConsoleInputProvider() HumanInputProvider {
	return &consoleInputProvider{}
}

func (p *consoleInputProvider) RequestInput(ctx context.Context, prompt string, options ...HumanInputOption) (*HumanInput, error) {
	config := HumanInputConfig{
		Timeout:  5 * time.Minute,
		Required: true,
	}

	for _, option := range options {
		option(&config)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	inputChan := make(chan *HumanInput, 1)
	errChan := make(chan error, 1)

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

// WorkflowBuilder implementation

func (b *workflowBuilder) AddNode(node WorkflowNode) WorkflowBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes[node.ID()] = node
	return b
}

func (b *workflowBuilder) AddEdge(from, to string) WorkflowBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.edges[from] = append(b.edges[from], to)
	return b
}

func (b *workflowBuilder) SetStart(nodeID string) WorkflowBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.startNode = nodeID
	return b
}

func (b *workflowBuilder) AddConditionalRoute(fromNodeID string, condition func(*WorkflowContext) bool, trueTarget, falseTarget string) WorkflowBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.conditionalRoutes[fromNodeID] = &conditionalRoute{
		condition:   condition,
		trueTarget:  trueTarget,
		falseTarget: falseTarget,
	}
	return b
}

func (b *workflowBuilder) WithCheckpointer(checkpointer Checkpointer) WorkflowBuilder {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.checkpointer = checkpointer
	return b
}

func (b *workflowBuilder) Execute(ctx context.Context, initialData map[string]any) (*WorkflowContext, error) {
	wfCtx := NewWorkflowContext(generateWorkflowID(), initialData)
	return b.executeFrom(ctx, b.startNode, wfCtx)
}

func (b *workflowBuilder) ExecuteWithCheckpoint(ctx context.Context, initialData map[string]any) (*WorkflowContext, error) {
	if b.checkpointer == nil {
		return nil, fmt.Errorf("no checkpointer configured")
	}

	wfCtx := NewWorkflowContext(generateWorkflowID(), initialData)

	// Create initial checkpoint
	checkpoint := CreateCheckpoint(
		wfCtx.ID,
		b.startNode,
		[]string{},
		wfCtx,
		CheckpointTypeAuto,
	)

	if err := b.checkpointer.Save(ctx, checkpoint); err != nil {
		return nil, fmt.Errorf("failed to save initial checkpoint: %w", err)
	}

	return b.executeFrom(ctx, b.startNode, wfCtx)
}

func (b *workflowBuilder) ResumeFromCheckpoint(ctx context.Context, workflowID string) (*WorkflowContext, error) {
	if b.checkpointer == nil {
		return nil, fmt.Errorf("no checkpointer configured")
	}

	checkpoint, err := b.checkpointer.Load(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	return b.executeFrom(ctx, checkpoint.CurrentNode, checkpoint.Context)
}

// Internal execution logic
func (b *workflowBuilder) executeFrom(ctx context.Context, nodeID string, wfCtx *WorkflowContext) (*WorkflowContext, error) {
	visited := make(map[string]bool)
	queue := []string{nodeID}

	for len(queue) > 0 {
		currentNodeID := queue[0]
		queue = queue[1:]

		if visited[currentNodeID] {
			continue
		}
		visited[currentNodeID] = true

		select {
		case <-ctx.Done():
			return wfCtx, ctx.Err()
		default:
		}

		node, exists := b.nodes[currentNodeID]
		if !exists {
			return wfCtx, fmt.Errorf("node %s not found", currentNodeID)
		}

		if err := node.Execute(ctx, wfCtx); err != nil {
			return wfCtx, fmt.Errorf("node %s execution failed: %w", currentNodeID, err)
		}

		// Check for conditional routing
		if route, hasRoute := b.conditionalRoutes[currentNodeID]; hasRoute {
			if route.condition(wfCtx) {
				queue = append(queue, route.trueTarget)
			} else {
				queue = append(queue, route.falseTarget)
			}
			continue
		}

		// Check if node specified next node
		if nextNode := wfCtx.Get("next_node"); nextNode != nil {
			if nextNodeStr, ok := nextNode.(string); ok && nextNodeStr != "" {
				wfCtx.Set("next_node", nil)
				queue = append(queue, nextNodeStr)
				continue
			}
		}

		// Add next nodes based on edges
		if nextNodes, exists := b.edges[currentNodeID]; exists {
			queue = append(queue, nextNodes...)
		}
	}

	return wfCtx, nil
}

// WorkflowContext implementation

func (c *WorkflowContext) Get(key string) any {
	return c.Data[key]
}

func (c *WorkflowContext) Set(key string, value any) {
	if c.Data == nil {
		c.Data = make(map[string]any)
	}
	c.Data[key] = value
}

func (c *WorkflowContext) AddMessage(role, content string) {
	c.Messages = append(c.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	})
}

func (c *WorkflowContext) GetMessages() []Message {
	messages := make([]Message, len(c.Messages))
	copy(messages, c.Messages)
	return messages
}

func (c *WorkflowContext) GetData() map[string]any {
	data := make(map[string]any)
	for k, v := range c.Data {
		data[k] = v
	}
	return data
}

// Node implementations

func (n *agentNode) ID() string {
	return n.id
}

func (n *agentNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	input := wfCtx.Get("input")
	if input == nil {
		input = "Continue with the workflow"
	}

	prompt := fmt.Sprintf("%v", input)
	if n.prompt != "" {
		prompt = fmt.Sprintf(n.prompt, input)
	}

	response, err := n.agent.Chat(ctx, prompt)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	wfCtx.Set("output", response)
	wfCtx.Set("last_agent", n.id)
	wfCtx.AddMessage(RoleAssistant, response)

	return nil
}

func (n *agentNode) WithPrompt(prompt string) WorkflowNode {
	newNode := *n
	newNode.prompt = prompt
	return &newNode
}

func (n *toolNode) ID() string {
	return n.id
}

func (n *toolNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	params := make(map[string]any)
	for k, v := range n.params {
		params[k] = v
	}

	if ctxParams := wfCtx.Get("tool_params"); ctxParams != nil {
		if ctxParamsMap, ok := ctxParams.(map[string]any); ok {
			for k, v := range ctxParamsMap {
				params[k] = v
			}
		}
	}

	result, err := n.tool.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("tool execution failed: %w", err)
	}

	wfCtx.Set("tool_result", result)
	wfCtx.Set("last_tool", n.id)

	return nil
}

func (n *toolNode) WithParams(params map[string]any) WorkflowNode {
	newNode := *n
	newNode.params = params
	return &newNode
}

func (n *parallelNode) ID() string {
	return n.id
}

func (n *parallelNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	if len(n.nodes) == 0 {
		return nil
	}

	errChan := make(chan error, len(n.nodes))

	for _, node := range n.nodes {
		go func(n WorkflowNode) {
			errChan <- n.Execute(ctx, wfCtx)
		}(node)
	}

	for i := 0; i < len(n.nodes); i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

func (n *conditionalNode) ID() string {
	return n.id
}

func (n *conditionalNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	for _, condition := range n.conditions {
		if condition.Check(wfCtx) {
			wfCtx.Set("next_node", condition.Target)
			return nil
		}
	}

	if n.defaultTo != "" {
		wfCtx.Set("next_node", n.defaultTo)
		return nil
	}

	return fmt.Errorf("no matching condition and no default route in conditional node %s", n.id)
}

func (n *conditionalNode) When(check func(*WorkflowContext) bool, target string) WorkflowNode {
	newNode := *n
	newNode.conditions = append(newNode.conditions, Condition{
		Check:  check,
		Target: target,
	})
	return &newNode
}

func (n *conditionalNode) Otherwise(target string) WorkflowNode {
	newNode := *n
	newNode.defaultTo = target
	return &newNode
}

func (n *humanNode) ID() string {
	return n.id
}

func (n *humanNode) Execute(ctx context.Context, wfCtx *WorkflowContext) error {
	prompt := n.prompt
	if contextData := wfCtx.Get("output"); contextData != nil {
		prompt = fmt.Sprintf("%s\n\nContext: %v", prompt, contextData)
	}

	input, err := n.provider.RequestInput(ctx, prompt, n.options...)
	if err != nil {
		return fmt.Errorf("human input failed: %w", err)
	}

	wfCtx.Set("human_input", input.Value)
	wfCtx.Set("human_data", input.Data)
	wfCtx.AddMessage("human", input.Value)

	return nil
}

func (n *humanNode) WithOptions(options ...HumanInputOption) WorkflowNode {
	newNode := *n
	newNode.options = append(newNode.options, options...)
	return &newNode
}

// Utility functions
func generateWorkflowID() string {
	return fmt.Sprintf("wf_%d", time.Now().UnixNano())
}

func When(check func(*WorkflowContext) bool, target string) Condition {
	return Condition{
		Check:  check,
		Target: target,
	}
}
