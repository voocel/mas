package tools

import (
	"sync"

	"github.com/voocel/mas/schema"
)

// Registry stores registered tools
type Registry struct {
	tools map[string]Tool
	mutex sync.RWMutex
}

// NewRegistry constructs a registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool
func (r *Registry) Register(tool Tool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := tool.Name()
	if name == "" {
		return schema.NewValidationError("tool.name", name, "tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return schema.NewToolError(name, "register", schema.ErrToolAlreadyExists)
	}

	r.tools[name] = tool
	return nil
}

// Unregister removes a tool
func (r *Registry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tools[name]; !exists {
		return schema.NewToolError(name, "unregister", schema.ErrToolNotFound)
	}

	delete(r.tools, name)
	return nil
}

// Get retrieves a tool
func (r *Registry) Get(name string) (Tool, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all tools
func (r *Registry) List() []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Names returns registered tool names
func (r *Registry) Names() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// Count returns the number of tools
func (r *Registry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.tools)
}

// Clear removes all tools
func (r *Registry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.tools = make(map[string]Tool)
}

// Has reports whether a tool exists
func (r *Registry) Has(name string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.tools[name]
	return exists
}

// GetByNames returns tools by name
func (r *Registry) GetByNames(names []string) []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tools := make([]Tool, 0, len(names))
	for _, name := range names {
		if tool, exists := r.tools[name]; exists {
			tools = append(tools, tool)
		}
	}
	return tools
}

// Filter returns tools that satisfy the predicate
func (r *Registry) Filter(predicate func(Tool) bool) []Tool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var filtered []Tool
	for _, tool := range r.tools {
		if predicate(tool) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// GetSchemas returns schemas for all tools
func (r *Registry) GetSchemas() map[string]*ToolSchema {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	schemas := make(map[string]*ToolSchema)
	for name, tool := range r.tools {
		schemas[name] = tool.Schema()
	}
	return schemas
}

// Clone duplicates the registry
func (r *Registry) Clone() *Registry {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	clone := NewRegistry()
	for name, tool := range r.tools {
		clone.tools[name] = tool
	}
	return clone
}

// Merge merges another registry
func (r *Registry) Merge(other *Registry) error {
	if other == nil {
		return nil
	}

	other.mutex.RLock()
	defer other.mutex.RUnlock()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	for name, tool := range other.tools {
		if _, exists := r.tools[name]; exists {
			return schema.NewToolError(name, "merge", schema.ErrToolAlreadyExists)
		}
		r.tools[name] = tool
	}

	return nil
}

// Global tool registry
var globalRegistry = NewRegistry()

// GlobalRegistry returns the global registry
func GlobalRegistry() *Registry {
	return globalRegistry
}

// Register adds a tool to the global registry
func Register(tool Tool) error {
	return globalRegistry.Register(tool)
}

// Get retrieves a tool from the global registry
func Get(name string) (Tool, bool) {
	return globalRegistry.Get(name)
}

// List returns all tools in the global registry
func List() []Tool {
	return globalRegistry.List()
}

// Has checks for a tool in the global registry
func Has(name string) bool {
	return globalRegistry.Has(name)
}
