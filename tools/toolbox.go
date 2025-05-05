package tools

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// Toolbox manages a collection of tools available to agents
type Toolbox struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewToolbox creates a new toolbox
func NewToolbox() *Toolbox {
	return &Toolbox{
		tools: make(map[string]Tool),
	}
}

// Add adds a tool to the toolbox
func (tb *Toolbox) Add(tool Tool) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (tb *Toolbox) Get(name string) (Tool, bool) {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	tool, ok := tb.tools[name]
	return tool, ok
}

// List lists all available tools
func (tb *Toolbox) List() []Tool {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	
	result := make([]Tool, 0, len(tb.tools))
	for _, tool := range tb.tools {
		result = append(result, tool)
	}
	return result
}

// Names gets all tool names
func (tb *Toolbox) Names() []string {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	
	names := make([]string, 0, len(tb.tools))
	for name := range tb.tools {
		names = append(names, name)
	}
	return names
}

// Execute executes a tool with the specified name
func (tb *Toolbox) Execute(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	tool, ok := tb.Get(toolName)
	if !ok {
		log.Printf("Tool not found in toolbox: %s", toolName)
		return nil, fmt.Errorf("%w: %s", ErrToolNotFound, toolName)
	}
	
	log.Printf("Toolbox executing tool: %s, params: %+v", toolName, params)
	
	result, err := tool.Execute(ctx, params)
	
	if err != nil {
		log.Printf("Toolbox failed to execute tool [%s]: %v", toolName, err)
		return nil, err
	}
	
	log.Printf("Toolbox successfully executed tool [%s]: %v", toolName, result)
	
	return result, nil
}

// Count returns the number of tools in the toolbox
func (tb *Toolbox) Count() int {
	tb.mu.RLock()
	defer tb.mu.RUnlock()
	return len(tb.tools)
}

// Clear clears the toolbox
func (tb *Toolbox) Clear() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.tools = make(map[string]Tool)
}

// WithTools creates a new toolbox with the provided tools
func WithTools(tools ...Tool) *Toolbox {
	tb := NewToolbox()
	for _, tool := range tools {
		tb.Add(tool)
	}
	return tb
} 