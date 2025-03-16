package agent

import (
	"context"
	"errors"

	"github.com/voocel/mas/knowledge"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/tools"
)

// Agent defines the basic interface for an agent
type Agent interface {
	// Name returns the agent's name
	Name() string

	// Perceive handles input information in the perception phase
	Perceive(ctx context.Context, input interface{}) error

	// Think processes information in the thinking phase
	Think(ctx context.Context) error

	// Act executes decisions and returns results in the action phase
	Act(ctx context.Context) (interface{}, error)

	// Process executes the full perceive-think-act cycle
	Process(ctx context.Context, input interface{}) (interface{}, error)

	// GetMemory gets the agent's memory system
	GetMemory() memory.Memory

	// GetKnowledgeGraph gets the agent's knowledge graph
	GetKnowledgeGraph() knowledge.Graph

	// GetTools gets the tools available to the agent
	GetTools() []tools.Tool
}

// Config contains configuration parameters for creating an agent
type Config struct {
	ID           string
	Name         string
	Description  string
	MemoryConfig memory.Config
	Tools        []tools.Tool
	LLMProvider  string
	LLMOptions   map[string]interface{}
}

// BaseAgent provides a base implementation of the Agent interface
type BaseAgent struct {
	name      string
	memory    memory.Memory
	knowledge knowledge.Graph
	tools     []tools.Tool
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(name string) *BaseAgent {
	return &BaseAgent{
		name: name,
	}
}

// NewBaseAgentWithOptions creates a new base agent with options
func NewBaseAgentWithOptions(name string, memory memory.Memory, kg knowledge.Graph, tools []tools.Tool) *BaseAgent {
	return &BaseAgent{
		name:      name,
		memory:    memory,
		knowledge: kg,
		tools:     tools,
	}
}

// Name gets the agent's name
func (a *BaseAgent) Name() string {
	return a.name
}

// GetMemory gets the agent's memory system
func (a *BaseAgent) GetMemory() memory.Memory {
	return a.memory
}

// GetKnowledgeGraph gets the agent's knowledge graph
func (a *BaseAgent) GetKnowledgeGraph() knowledge.Graph {
	return a.knowledge
}

// GetTools gets the tools available to the agent
func (a *BaseAgent) GetTools() []tools.Tool {
	return a.tools
}

// Perceive default implementation - not implemented
func (a *BaseAgent) Perceive(ctx context.Context, input interface{}) error {
	return errors.New("unimplemented method: Perceive is not available for base agent")
}

// Think default implementation - not implemented
func (a *BaseAgent) Think(ctx context.Context) error {
	return errors.New("unimplemented method: Think is not available for base agent")
}

// Act default implementation - not implemented
func (a *BaseAgent) Act(ctx context.Context) (interface{}, error) {
	return nil, errors.New("unimplemented method: Act is not available for base agent")
}

// Process executes the full perceive-think-act cycle
func (a *BaseAgent) Process(ctx context.Context, input interface{}) (interface{}, error) {
	if err := a.Perceive(ctx, input); err != nil {
		return nil, err
	}

	if err := a.Think(ctx); err != nil {
		return nil, err
	}

	return a.Act(ctx)
}
