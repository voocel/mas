package agent

import (
	"fmt"
	"sync"
)

// Registry provides agent registration and management functionality
type Registry struct {
	agents map[string]Agent
	mu     sync.RWMutex
}

// NewRegistry creates a new agent registry
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]Agent),
	}
}

// Register registers an agent
func (r *Registry) Register(role string, agent Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[role] = agent
}

// Get retrieves an agent by role
func (r *Registry) Get(role string) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	agent, ok := r.agents[role]
	return agent, ok
}

// List lists all registered agents
func (r *Registry) List() map[string]Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// create a copy to avoid concurrency issues
	result := make(map[string]Agent, len(r.agents))
	for k, v := range r.agents {
		result[k] = v
	}
	return result
}

// RequireRoles ensures all specified roles are registered
func (r *Registry) RequireRoles(roles ...string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var missing []string
	for _, role := range roles {
		if _, ok := r.agents[role]; !ok {
			missing = append(missing, role)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required agent roles: %v", missing)
	}

	return nil
}

// HasRole checks if a specific role is registered
func (r *Registry) HasRole(role string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.agents[role]
	return ok
}

// Clear clears all registered agents
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents = make(map[string]Agent)
}
