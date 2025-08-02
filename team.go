package mas

import (
	"context"
	"fmt"
	"sync"
)

// Team represents a collection of agents working together
type Team interface {
	// Add adds an agent to the team
	Add(name string, agent Agent) Team

	// Remove removes an agent from the team
	Remove(name string) Team

	// Execute runs a task through the team
	Execute(ctx context.Context, input string) (string, error)

	// WithFlow defines the execution flow between agents
	WithFlow(flow ...string) Team

	// GetAgent retrieves an agent by name
	GetAgent(name string) (Agent, bool)

	// ListAgents returns all agent names
	ListAgents() []string

	// SetSharedMemory sets a shared memory for all agents
	SetSharedMemory(memory Memory) Team
}

// TeamConfig contains configuration for a team
type TeamConfig struct {
	Name         string
	Description  string
	DefaultFlow  []string
	SharedMemory Memory
	Parallel     bool
}

// DefaultTeamConfig returns a default team configuration
func DefaultTeamConfig() TeamConfig {
	return TeamConfig{
		Name:        "team",
		Description: "A team of agents",
		DefaultFlow: []string{},
		Parallel:    false,
	}
}

// team implements the Team interface
type team struct {
	config TeamConfig
	agents map[string]Agent
	flow   []string
	mu     sync.RWMutex
}

// NewTeam creates a new team with default configuration
func NewTeam() Team {
	return NewTeamWithConfig(DefaultTeamConfig())
}

// NewTeamWithConfig creates a new team with custom configuration
func NewTeamWithConfig(config TeamConfig) Team {
	return &team{
		config: config,
		agents: make(map[string]Agent),
		flow:   config.DefaultFlow,
	}
}

// Add adds an agent to the team
func (t *team) Add(name string, agent Agent) Team {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create a new team instance to maintain immutability
	newTeam := &team{
		config: t.config,
		agents: make(map[string]Agent),
		flow:   make([]string, len(t.flow)),
		mu:     sync.RWMutex{},
	}

	// Copy existing agents
	for k, v := range t.agents {
		newTeam.agents[k] = v
	}
	copy(newTeam.flow, t.flow)

	// Add the new agent
	newTeam.agents[name] = agent

	// If shared memory is configured, set it for the agent
	if t.config.SharedMemory != nil {
		newTeam.agents[name] = agent.WithMemory(t.config.SharedMemory)
	}

	return newTeam
}

// Remove removes an agent from the team
func (t *team) Remove(name string) Team {
	t.mu.Lock()
	defer t.mu.Unlock()

	newTeam := &team{
		config: t.config,
		agents: make(map[string]Agent),
		flow:   make([]string, len(t.flow)),
		mu:     sync.RWMutex{},
	}

	// Copy existing agents except the one to remove
	for k, v := range t.agents {
		if k != name {
			newTeam.agents[k] = v
		}
	}
	copy(newTeam.flow, t.flow)

	// Remove from flow if present
	var newFlow []string
	for _, agentName := range newTeam.flow {
		if agentName != name {
			newFlow = append(newFlow, agentName)
		}
	}
	newTeam.flow = newFlow

	return newTeam
}

// Execute runs a task through the team
func (t *team) Execute(ctx context.Context, input string) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.agents) == 0 {
		return "", fmt.Errorf("no agents in team")
	}

	// If no flow is defined, use the first agent
	if len(t.flow) == 0 {
		for _, agent := range t.agents {
			return agent.Chat(ctx, input)
		}
	}

	// Execute according to the defined flow
	currentInput := input
	var result string
	var err error

	if t.config.Parallel {
		// Parallel execution
		result, err = t.executeParallel(ctx, currentInput)
	} else {
		// Sequential execution
		result, err = t.executeSequential(ctx, currentInput)
	}

	return result, err
}

// executeSequential executes agents sequentially according to the flow
func (t *team) executeSequential(ctx context.Context, input string) (string, error) {
	currentInput := input

	for _, agentName := range t.flow {
		agent, exists := t.agents[agentName]
		if !exists {
			return "", fmt.Errorf("agent '%s' not found in team", agentName)
		}

		result, err := agent.Chat(ctx, currentInput)
		if err != nil {
			return "", fmt.Errorf("error from agent '%s': %w", agentName, err)
		}

		currentInput = result
	}

	return currentInput, nil
}

// executeParallel executes agents in parallel and combines results
func (t *team) executeParallel(ctx context.Context, input string) (string, error) {
	type agentResult struct {
		name   string
		result string
		err    error
	}

	resultChan := make(chan agentResult, len(t.flow))

	// Start all agents in parallel
	for _, agentName := range t.flow {
		go func(name string) {
			agent, exists := t.agents[name]
			if !exists {
				resultChan <- agentResult{
					name: name,
					err:  fmt.Errorf("agent '%s' not found in team", name),
				}
				return
			}

			result, err := agent.Chat(ctx, input)
			resultChan <- agentResult{
				name:   name,
				result: result,
				err:    err,
			}
		}(agentName)
	}

	// Collect results
	var results []string
	var errors []error

	for i := 0; i < len(t.flow); i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case result := <-resultChan:
			if result.err != nil {
				errors = append(errors, fmt.Errorf("error from agent '%s': %w", result.name, result.err))
			} else {
				results = append(results, fmt.Sprintf("[%s]: %s", result.name, result.result))
			}
		}
	}

	if len(errors) > 0 {
		return "", fmt.Errorf("errors from agents: %v", errors)
	}

	// Combine all results
	combinedResult := ""
	for _, result := range results {
		if combinedResult != "" {
			combinedResult += "\n\n"
		}
		combinedResult += result
	}

	return combinedResult, nil
}

// WithFlow defines the execution flow between agents
func (t *team) WithFlow(flow ...string) Team {
	t.mu.Lock()
	defer t.mu.Unlock()

	newTeam := &team{
		config: t.config,
		agents: make(map[string]Agent),
		flow:   make([]string, len(flow)),
		mu:     sync.RWMutex{},
	}

	// Copy existing agents
	for k, v := range t.agents {
		newTeam.agents[k] = v
	}
	copy(newTeam.flow, flow)

	return newTeam
}

// GetAgent retrieves an agent by name
func (t *team) GetAgent(name string) (Agent, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	agent, exists := t.agents[name]
	return agent, exists
}

// ListAgents returns all agent names
func (t *team) ListAgents() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	names := make([]string, 0, len(t.agents))
	for name := range t.agents {
		names = append(names, name)
	}
	return names
}

// SetSharedMemory sets a shared memory for all agents
func (t *team) SetSharedMemory(memory Memory) Team {
	t.mu.Lock()
	defer t.mu.Unlock()

	newConfig := t.config
	newConfig.SharedMemory = memory

	newTeam := &team{
		config: newConfig,
		agents: make(map[string]Agent),
		flow:   make([]string, len(t.flow)),
		mu:     sync.RWMutex{},
	}

	// Copy agents and apply shared memory
	for k, v := range t.agents {
		newTeam.agents[k] = v.WithMemory(memory)
	}
	copy(newTeam.flow, t.flow)

	return newTeam
}

// WithParallel configures the team to execute agents in parallel
func (t *team) WithParallel(parallel bool) Team {
	t.mu.Lock()
	defer t.mu.Unlock()

	newConfig := t.config
	newConfig.Parallel = parallel

	newTeam := &team{
		config: newConfig,
		agents: make(map[string]Agent),
		flow:   make([]string, len(t.flow)),
		mu:     sync.RWMutex{},
	}

	// Copy existing agents
	for k, v := range t.agents {
		newTeam.agents[k] = v
	}
	copy(newTeam.flow, t.flow)

	return newTeam
}

// Count returns the number of agents in the team
func (t *team) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.agents)
}

// IsEmpty returns true if the team has no agents
func (t *team) IsEmpty() bool {
	return t.Count() == 0
}

// HasAgent returns true if the team contains an agent with the given name
func (t *team) HasAgent(name string) bool {
	_, exists := t.GetAgent(name)
	return exists
}