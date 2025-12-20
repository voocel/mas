package multi

import (
	"fmt"
	"sort"
	"sync"

	"github.com/voocel/mas/agent"
)

// Team is a simple multi-agent registry.
type Team struct {
	mu     sync.RWMutex
	agents map[string]*agent.Agent
}

// NewTeam creates an empty team.
func NewTeam() *Team {
	return &Team{agents: make(map[string]*agent.Agent)}
}

// Add registers an agent.
func (t *Team) Add(name string, ag *agent.Agent) error {
	if name == "" {
		return fmt.Errorf("team: name cannot be empty")
	}
	if ag == nil {
		return fmt.Errorf("team: agent is nil")
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.agents[name]; exists {
		return fmt.Errorf("team: agent %s already exists", name)
	}
	t.agents[name] = ag
	return nil
}

// Get retrieves an agent by name.
func (t *Team) Get(name string) (*agent.Agent, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	ag, ok := t.agents[name]
	return ag, ok
}

// Remove deletes an agent by name.
func (t *Team) Remove(name string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.agents[name]; !ok {
		return fmt.Errorf("team: agent %s not found", name)
	}
	delete(t.agents, name)
	return nil
}

// List returns sorted agent names.
func (t *Team) List() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	names := make([]string, 0, len(t.agents))
	for name := range t.agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Route routes to an agent by name.
func (t *Team) Route(name string) (*agent.Agent, error) {
	ag, ok := t.Get(name)
	if !ok {
		return nil, fmt.Errorf("team: agent %s not found", name)
	}
	return ag, nil
}
