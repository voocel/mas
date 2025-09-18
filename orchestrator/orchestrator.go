package orchestrator

import (
	"sync"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/workflows"
)

// Orchestrator defines the orchestrator interface.
type Orchestrator interface {
	// AddAgent adds an agent.
	AddAgent(name string, agent agent.Agent) error

	// AddWorkflow adds a workflow.
	AddWorkflow(name string, workflow workflows.Workflow) error

	// Execute executes a request.
	Execute(ctx runtime.Context, request ExecuteRequest) (ExecuteResponse, error)

	// ExecuteStream executes a streaming request.
	ExecuteStream(ctx runtime.Context, request ExecuteRequest) (<-chan schema.StreamEvent, error)

	// GetAgent gets an agent.
	GetAgent(name string) (agent.Agent, bool)

	// ListAgents lists all agents.
	ListAgents() []string

	// RemoveAgent removes an agent.
	RemoveAgent(name string) error
}

// ExecuteRequest is an execution request.
type ExecuteRequest struct {
	Input      schema.Message         `json:"input"`
	Target     string                 `json:"target"`     // Target agent or workflow name.
	Type       ExecuteType            `json:"type"`       // Execution type.
	Parameters map[string]interface{} `json:"parameters"` // Additional parameters.
	Metadata   map[string]interface{} `json:"metadata"`   // Metadata.
}

// ExecuteResponse is an execution response.
type ExecuteResponse struct {
	Output   schema.Message         `json:"output"`
	Source   string                 `json:"source"`   // Response source.
	Metadata map[string]interface{} `json:"metadata"` // Metadata.
	Trace    []ExecutionStep        `json:"trace"`    // Execution trace.
}

// ExecuteType is the execution type.
type ExecuteType string

const (
	ExecuteTypeAgent    ExecuteType = "agent"
	ExecuteTypeWorkflow ExecuteType = "workflow"
	ExecuteTypeAuto     ExecuteType = "auto"
)

// ExecutionStep is an execution step.
type ExecutionStep struct {
	Agent    string                 `json:"agent"`
	Input    schema.Message         `json:"input"`
	Output   schema.Message         `json:"output"`
	Duration int64                  `json:"duration"` // Milliseconds.
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// BaseOrchestrator is the base orchestrator implementation.
type BaseOrchestrator struct {
	agents    map[string]agent.Agent
	workflows map[string]workflows.Workflow
	mutex     sync.RWMutex
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator() *BaseOrchestrator {
	return &BaseOrchestrator{
		agents:    make(map[string]agent.Agent),
		workflows: make(map[string]workflows.Workflow),
	}
}

func (o *BaseOrchestrator) AddAgent(name string, ag agent.Agent) error {
	if name == "" {
		return schema.NewValidationError("name", name, "agent name cannot be empty")
	}
	if ag == nil {
		return schema.NewValidationError("agent", ag, "agent cannot be nil")
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if _, exists := o.agents[name]; exists {
		return schema.NewAgentError(name, "add", schema.ErrAgentAlreadyExists)
	}

	o.agents[name] = ag
	return nil
}

func (o *BaseOrchestrator) AddWorkflow(name string, workflow workflows.Workflow) error {
	if name == "" {
		return schema.NewValidationError("name", name, "workflow name cannot be empty")
	}
	if workflow == nil {
		return schema.NewValidationError("workflow", workflow, "workflow cannot be nil")
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if _, exists := o.workflows[name]; exists {
		return schema.NewWorkflowError(name, "add", schema.ErrWorkflowNotFound)
	}

	o.workflows[name] = workflow
	return nil
}

func (o *BaseOrchestrator) GetAgent(name string) (agent.Agent, bool) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	ag, exists := o.agents[name]
	return ag, exists
}

func (o *BaseOrchestrator) ListAgents() []string {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	names := make([]string, 0, len(o.agents))
	for name := range o.agents {
		names = append(names, name)
	}
	return names
}

func (o *BaseOrchestrator) RemoveAgent(name string) error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if _, exists := o.agents[name]; !exists {
		return schema.NewAgentError(name, "remove", schema.ErrAgentNotFound)
	}

	delete(o.agents, name)
	return nil
}

func (o *BaseOrchestrator) Execute(ctx runtime.Context, request ExecuteRequest) (ExecuteResponse, error) {
	switch request.Type {
	case ExecuteTypeAgent:
		return o.executeAgent(ctx, request)
	case ExecuteTypeWorkflow:
		return o.executeWorkflow(ctx, request)
	case ExecuteTypeAuto:
		return o.executeAuto(ctx, request)
	default:
		return ExecuteResponse{}, schema.NewValidationError("type", request.Type, "unsupported execute type")
	}
}

func (o *BaseOrchestrator) ExecuteStream(ctx runtime.Context, request ExecuteRequest) (<-chan schema.StreamEvent, error) {
	switch request.Type {
	case ExecuteTypeAgent:
		return o.executeAgentStream(ctx, request)
	case ExecuteTypeWorkflow:
		return o.executeWorkflowStream(ctx, request)
	case ExecuteTypeAuto:
		return o.executeAutoStream(ctx, request)
	default:
		return nil, schema.NewValidationError("type", request.Type, "unsupported execute type")
	}
}

// executeAgent executes a single agent.
func (o *BaseOrchestrator) executeAgent(ctx runtime.Context, request ExecuteRequest) (ExecuteResponse, error) {
	ag, exists := o.GetAgent(request.Target)
	if !exists {
		return ExecuteResponse{}, schema.NewAgentError(request.Target, "execute", schema.ErrAgentNotFound)
	}

	// Execute the agent.
	output, err := ag.Execute(ctx, request.Input)
	if err != nil {
		return ExecuteResponse{}, err
	}

	response := ExecuteResponse{
		Output: output,
		Source: request.Target,
		Trace: []ExecutionStep{
			{
				Agent:  request.Target,
				Input:  request.Input,
				Output: output,
			},
		},
	}

	return response, nil
}

// executeWorkflow executes a workflow.
func (o *BaseOrchestrator) executeWorkflow(ctx runtime.Context, request ExecuteRequest) (ExecuteResponse, error) {
	o.mutex.RLock()
	workflow, exists := o.workflows[request.Target]
	o.mutex.RUnlock()

	if !exists {
		return ExecuteResponse{}, schema.NewWorkflowError(request.Target, "execute", schema.ErrWorkflowNotFound)
	}

	// Execute the workflow.
	output, err := workflow.Execute(ctx, request.Input)
	if err != nil {
		return ExecuteResponse{}, err
	}

	// Build the response.
	response := ExecuteResponse{
		Output: output,
		Source: request.Target,
		// TODO: Get the execution trace from the workflow.
	}

	return response, nil
}

// executeAuto automatically selects the execution method.
func (o *BaseOrchestrator) executeAuto(ctx runtime.Context, request ExecuteRequest) (ExecuteResponse, error) {
	// Try agent first.
	if _, exists := o.GetAgent(request.Target); exists {
		request.Type = ExecuteTypeAgent
		return o.executeAgent(ctx, request)
	}

	// Then try workflow.
	o.mutex.RLock()
	_, exists := o.workflows[request.Target]
	o.mutex.RUnlock()

	if exists {
		request.Type = ExecuteTypeWorkflow
		return o.executeWorkflow(ctx, request)
	}

	return ExecuteResponse{}, schema.NewValidationError("target", request.Target, "target not found")
}

// executeAgentStream executes an agent in streaming mode.
func (o *BaseOrchestrator) executeAgentStream(ctx runtime.Context, request ExecuteRequest) (<-chan schema.StreamEvent, error) {
	ag, exists := o.GetAgent(request.Target)
	if !exists {
		return nil, schema.NewAgentError(request.Target, "execute_stream", schema.ErrAgentNotFound)
	}

	return ag.ExecuteStream(ctx, request.Input)
}

// executeWorkflowStream executes a workflow in streaming mode.
func (o *BaseOrchestrator) executeWorkflowStream(ctx runtime.Context, request ExecuteRequest) (<-chan schema.StreamEvent, error) {
	o.mutex.RLock()
	workflow, exists := o.workflows[request.Target]
	o.mutex.RUnlock()

	if !exists {
		return nil, schema.NewWorkflowError(request.Target, "execute_stream", schema.ErrWorkflowNotFound)
	}

	return workflow.ExecuteStream(ctx, request.Input)
}

// executeAutoStream automatically selects the streaming execution method.
func (o *BaseOrchestrator) executeAutoStream(ctx runtime.Context, request ExecuteRequest) (<-chan schema.StreamEvent, error) {
	// Try agent first.
	if ag, exists := o.GetAgent(request.Target); exists {
		return ag.ExecuteStream(ctx, request.Input)
	}

	// Then try workflow.
	o.mutex.RLock()
	workflow, exists := o.workflows[request.Target]
	o.mutex.RUnlock()

	if exists {
		return workflow.ExecuteStream(ctx, request.Input)
	}

	return nil, schema.NewValidationError("target", request.Target, "target not found")
}
