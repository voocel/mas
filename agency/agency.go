package agency

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/orchestrator"
)

// Agency represents a group of collaborating agents and their workflows
type Agency struct {
	// Name
	Name string

	// Collection of agents
	Agents map[string]agent.Agent

	// Flow chart - defines communication relationships between agents
	FlowChart *FlowChart

	// Orchestrator
	Orchestrator orchestrator.Orchestrator

	// Shared state
	SharedState map[string]interface{}

	// Shared instructions
	SharedInstructions string

	// Mutex, protects shared state
	mu sync.RWMutex
}

// Config configures an Agency instance
type Config struct {
	// Name
	Name string

	// Shared instructions
	SharedInstructions string

	// Orchestrator instance
	Orchestrator orchestrator.Orchestrator

	// Default model
	DefaultModel string

	// Temperature parameter
	Temperature float64

	// Maximum tokens to generate
	MaxTokens int
}

// New creates a new Agency
func New(config Config) *Agency {
	agency := &Agency{
		Name:               config.Name,
		Agents:             make(map[string]agent.Agent),
		FlowChart:          NewFlowChart(),
		SharedState:        make(map[string]interface{}),
		SharedInstructions: config.SharedInstructions,
	}

	// Use provided orchestrator or create a default one
	if config.Orchestrator != nil {
		agency.Orchestrator = config.Orchestrator
	} else {
		agency.Orchestrator = orchestrator.NewBasicOrchestrator(orchestrator.Options{})
	}

	return agency
}

// AddAgent adds an agent to the Agency
func (a *Agency) AddAgent(agent agent.Agent) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	agentID := agent.Name()
	if _, exists := a.Agents[agentID]; exists {
		return fmt.Errorf("agent with ID %s already exists", agentID)
	}

	a.Agents[agentID] = agent

	// Also register with the orchestrator
	err := a.Orchestrator.RegisterAgent(agent)
	if err != nil {
		delete(a.Agents, agentID)
		return err
	}

	return nil
}

// GetAgent gets an agent by specified ID
func (a *Agency) GetAgent(id string) (agent.Agent, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agent, exists := a.Agents[id]
	if !exists {
		return nil, fmt.Errorf("agent with ID %s not found", id)
	}

	return agent, nil
}

// ListAgents lists all agents
func (a *Agency) ListAgents() []agent.Agent {
	a.mu.RLock()
	defer a.mu.RUnlock()

	agents := make([]agent.Agent, 0, len(a.Agents))
	for _, agent := range a.Agents {
		agents = append(agents, agent)
	}

	return agents
}

// SetFlowChart sets the communication flow chart
func (a *Agency) SetFlowChart(flowChart *FlowChart) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.FlowChart = flowChart
}

// DefineFlowChart defines communication relationships through Flow
func (a *Agency) DefineFlowChart(flows []Flow) error {
	flowChart := NewFlowChart()

	// First element is the entry point
	for _, flow := range flows {
		if len(flow) == 1 {
			flowChart.AddEntryPoint(flow[0].Name())
		} else if len(flow) == 2 {
			// Define connection from first to second
			flowChart.AddConnection(flow[0].Name(), flow[1].Name())
		} else {
			return fmt.Errorf("invalid flow definition: each flow must contain 1 or 2 agents")
		}
	}

	a.SetFlowChart(flowChart)
	return nil
}

// Execute executes a task
func (a *Agency) Execute(ctx context.Context, input string) (string, error) {
	if len(a.FlowChart.EntryPoints) == 0 {
		return "", fmt.Errorf("no entry point defined in the agency")
	}

	// Use the first entry point agent
	entryAgentID := a.FlowChart.EntryPoints[0]
	_, err := a.GetAgent(entryAgentID)
	if err != nil {
		return "", err
	}

	// Create task
	task := orchestrator.Task{
		Name:        "Execute Agency Task",
		Description: fmt.Sprintf("Process input via %s", entryAgentID),
		AgentIDs:    []string{entryAgentID},
		Input:       input,
	}

	// Submit task to orchestrator
	taskID, err := a.Orchestrator.SubmitTask(ctx, task)
	if err != nil {
		return "", err
	}

	// Wait for task completion
	for {
		task, err := a.Orchestrator.GetTask(taskID)
		if err != nil {
			return "", err
		}

		if task.Status == orchestrator.TaskStatusCompleted {
			result, ok := task.Output.(string)
			if !ok {
				return "", fmt.Errorf("task output is not a string")
			}
			return result, nil
		} else if task.Status == orchestrator.TaskStatusFailed {
			return "", fmt.Errorf("task failed: %s", task.Error)
		}

		// Wait for a while before checking again
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue checking
		}
	}
}

// RegisterWorkflow registers a workflow with the orchestrator
func (a *Agency) RegisterWorkflow(workflow *Workflow) error {
	// To be implemented, requires extending the orchestrator package to support workflows
	return fmt.Errorf("workflow support not yet implemented")
}

// Flow represents a communication connection relationship, containing 1 or 2 agents
type Flow []agent.Agent
