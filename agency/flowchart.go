package agency

import (
	"sync"
)

// FlowChart defines communication and collaboration relationships between agents
type FlowChart struct {
	// Connections from sender to receiver
	Connections map[string]map[string]bool

	// Entry agents - entry points for direct user interaction
	EntryPoints []string

	// Mutex protects concurrent access
	mu sync.RWMutex
}

// NewFlowChart creates a new FlowChart
func NewFlowChart() *FlowChart {
	return &FlowChart{
		Connections: make(map[string]map[string]bool),
		EntryPoints: make([]string, 0),
	}
}

// AddConnection adds a communication connection from sender to receiver
func (fc *FlowChart) AddConnection(from, to string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if _, exists := fc.Connections[from]; !exists {
		fc.Connections[from] = make(map[string]bool)
	}
	fc.Connections[from][to] = true
}

// RemoveConnection removes a communication connection from sender to receiver
func (fc *FlowChart) RemoveConnection(from, to string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if receivers, exists := fc.Connections[from]; exists {
		delete(receivers, to)

		// If sender has no more receivers, remove the entire mapping
		if len(receivers) == 0 {
			delete(fc.Connections, from)
		}
	}
}

// CanCommunicate checks if the sender can communicate with the receiver
func (fc *FlowChart) CanCommunicate(from, to string) bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if receivers, exists := fc.Connections[from]; exists {
		return receivers[to]
	}
	return false
}

// GetReceivers gets all receivers that the sender can communicate with
func (fc *FlowChart) GetReceivers(from string) []string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	receivers := make([]string, 0)
	if receiversMap, exists := fc.Connections[from]; exists {
		for receiver := range receiversMap {
			receivers = append(receivers, receiver)
		}
	}
	return receivers
}

// AddEntryPoint adds an entry point agent
func (fc *FlowChart) AddEntryPoint(agentID string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	// Check if it already exists
	for _, ep := range fc.EntryPoints {
		if ep == agentID {
			return
		}
	}
	fc.EntryPoints = append(fc.EntryPoints, agentID)
}

// RemoveEntryPoint removes an entry point agent
func (fc *FlowChart) RemoveEntryPoint(agentID string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	for i, ep := range fc.EntryPoints {
		if ep == agentID {
			// Remove from slice
			fc.EntryPoints = append(fc.EntryPoints[:i], fc.EntryPoints[i+1:]...)
			return
		}
	}
}

// IsEntryPoint checks if an agent is an entry point
func (fc *FlowChart) IsEntryPoint(agentID string) bool {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	for _, ep := range fc.EntryPoints {
		if ep == agentID {
			return true
		}
	}
	return false
} 