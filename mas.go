package mas

import (
	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/communication"
	"github.com/voocel/mas/orchestrator"
	"github.com/voocel/mas/tools"
)

// System represents a complete multi-agent system
type System struct {
	Registry     *agent.Registry
	Orchestrator orchestrator.Orchestrator
	Bus          communication.Bus
	ToolRegistry *tools.Registry
}

// SystemConfig contains configuration options for creating the system
type SystemConfig struct {
	EnableMemoryBus    bool
	EnableOrchestrator bool
}

// NewSystem creates a new multi-agent system
func NewSystem(config SystemConfig) *System {
	system := &System{
		Registry: agent.NewRegistry(),
	}

	// Initialize communication bus
	if config.EnableMemoryBus {
		system.Bus = communication.NewMemoryBus(communication.Config{
			Type:       "memory",
			BufferSize: 100,
		})
	}

	// Initialize orchestrator
	if config.EnableOrchestrator {
		system.Orchestrator = orchestrator.NewBasicOrchestrator(orchestrator.Options{
			Bus: system.Bus,
		})
	}

	// Initialize tool registry
	system.ToolRegistry = tools.NewRegistry()

	return system
}

// RegisterAgent registers an agent with a specific role
func (s *System) RegisterAgent(role string, agent agent.Agent) {
	s.Registry.Register(role, agent)

	// If orchestrator is enabled, also register in orchestrator
	if s.Orchestrator != nil {
		s.Orchestrator.RegisterAgent(agent)
	}
}

// RequireRoles ensures all required roles are registered
func (s *System) RequireRoles(roles ...string) error {
	return s.Registry.RequireRoles(roles...)
}

// GetAgent retrieves the agent for a specific role
func (s *System) GetAgent(role string) (agent.Agent, bool) {
	return s.Registry.Get(role)
}

// ListAgents lists all registered agents
func (s *System) ListAgents() map[string]agent.Agent {
	return s.Registry.List()
}

// DefaultSystem creates a multi-agent system with default configuration
func DefaultSystem() *System {
	return NewSystem(SystemConfig{
		EnableMemoryBus:    true,
		EnableOrchestrator: true,
	})
}
