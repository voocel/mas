package engine

import (
	"fmt"
	"strings"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// StateManager manages context state operations
type StateManager struct {
	engine *ContextEngine
}

// NewStateManager creates a new state manager
func NewStateManager(engine *ContextEngine) *StateManager {
	return &StateManager{
		engine: engine,
	}
}

// CalculateTokenCount estimates token count for a context state
func (sm *StateManager) CalculateTokenCount(state *contextpkg.ContextState) int {
	totalTokens := 0

	// Count message tokens
	for _, msg := range state.Messages {
		totalTokens += sm.estimateTokens(msg.Content)
	}

	// Count scratchpad tokens
	for _, v := range state.Scratchpad {
		if str, ok := v.(string); ok {
			totalTokens += sm.estimateTokens(str)
		}
	}

	// Count selected data tokens
	for _, v := range state.SelectedData {
		if str, ok := v.(string); ok {
			totalTokens += sm.estimateTokens(str)
		}
	}

	// Count compressed context tokens
	if state.CompressedCtx != nil {
		totalTokens += sm.estimateTokens(state.CompressedCtx.Summary)
		for _, point := range state.CompressedCtx.KeyPoints {
			totalTokens += sm.estimateTokens(point)
		}
	}

	state.TokenCount = totalTokens
	return totalTokens
}

// estimateTokens provides a rough token count estimation
func (sm *StateManager) estimateTokens(text string) int {
	// Rough estimation: 1 token ≈ 4 characters for English text
	return len(text) / 4
}

// AnalyzeState analyzes the current context state
func (sm *StateManager) AnalyzeState(state *contextpkg.ContextState) *contextpkg.StateAnalysis {
	tokenCount := sm.CalculateTokenCount(state)
	messageCount := len(state.Messages)

	// Calculate complexity based on various factors
	complexity := sm.calculateComplexity(state)

	// Calculate memory pressure
	memoryPressure := sm.calculateMemoryPressure(state)

	// Count recent activity (messages in last 10 minutes)
	recentActivity := sm.countRecentActivity(state, 10*time.Minute)

	return &contextpkg.StateAnalysis{
		TokenCount:     tokenCount,
		MessageCount:   messageCount,
		AgentCount:     sm.countActiveAgents(state),
		Complexity:     complexity,
		MemoryPressure: memoryPressure,
		RecentActivity: recentActivity,
	}
}

// calculateComplexity calculates the complexity score of the context
func (sm *StateManager) calculateComplexity(state *contextpkg.ContextState) float64 {
	complexity := 0.0

	// Factor 1: Number of messages
	complexity += float64(len(state.Messages)) * 0.1

	// Factor 2: Average message length
	if len(state.Messages) > 0 {
		totalLength := 0
		for _, msg := range state.Messages {
			totalLength += len(msg.Content)
		}
		avgLength := float64(totalLength) / float64(len(state.Messages))
		complexity += avgLength / 1000 // Normalize
	}

	// Factor 3: Scratchpad complexity
	complexity += float64(len(state.Scratchpad)) * 0.2

	// Factor 4: Selected data complexity
	complexity += float64(len(state.SelectedData)) * 0.15

	// Factor 5: Isolated context complexity
	complexity += float64(len(state.IsolatedCtx)) * 0.1

	// Normalize to 0-1 range
	if complexity > 10 {
		complexity = 10
	}
	return complexity / 10
}

// calculateMemoryPressure calculates memory pressure based on token count
func (sm *StateManager) calculateMemoryPressure(state *contextpkg.ContextState) float64 {
	maxTokens := float64(sm.engine.config.MaxTokens)
	currentTokens := float64(state.TokenCount)

	pressure := currentTokens / maxTokens
	if pressure > 1.0 {
		pressure = 1.0
	}

	return pressure
}

// countActiveAgents counts the number of active agents in the context
func (sm *StateManager) countActiveAgents(state *contextpkg.ContextState) int {
	agents := make(map[string]bool)

	// Count agents from messages
	for _, msg := range state.Messages {
		if msg.Name != "" {
			agents[msg.Name] = true
		}
	}

	// Count current agent
	if state.AgentID != "" {
		agents[state.AgentID] = true
	}

	return len(agents)
}

// countRecentActivity counts recent activity within the specified duration
func (sm *StateManager) countRecentActivity(state *contextpkg.ContextState, duration time.Duration) int {
	cutoff := time.Now().Add(-duration)
	count := 0

	for _, msg := range state.Messages {
		if msg.Timestamp.After(cutoff) {
			count++
		}
	}

	return count
}

// MergeStates merges multiple context states
func (sm *StateManager) MergeStates(states ...*contextpkg.ContextState) (*contextpkg.ContextState, error) {
	if len(states) == 0 {
		return nil, fmt.Errorf("no states to merge")
	}

	if len(states) == 1 {
		return states[0].Copy(), nil
	}

	// Use the first state as base
	merged := states[0].Copy()

	// Merge messages from all states
	allMessages := make([]contextpkg.Message, 0)
	allMessages = append(allMessages, merged.Messages...)

	for i := 1; i < len(states); i++ {
		allMessages = append(allMessages, states[i].Messages...)
	}

	// Sort messages by timestamp
	merged.Messages = sm.sortMessagesByTimestamp(allMessages)

	// Merge scratchpads
	for i := 1; i < len(states); i++ {
		for k, v := range states[i].Scratchpad {
			merged.Scratchpad[k] = v
		}
	}

	// Merge selected data
	for i := 1; i < len(states); i++ {
		for k, v := range states[i].SelectedData {
			merged.SelectedData[k] = v
		}
	}

	// Update metadata
	merged.Timestamp = time.Now()
	sm.CalculateTokenCount(merged)

	return merged, nil
}

// sortMessagesByTimestamp sorts messages by timestamp
func (sm *StateManager) sortMessagesByTimestamp(messages []contextpkg.Message) []contextpkg.Message {
	// Simple bubble sort for now - could be optimized
	for i := 0; i < len(messages)-1; i++ {
		for j := 0; j < len(messages)-i-1; j++ {
			if messages[j].Timestamp.After(messages[j+1].Timestamp) {
				messages[j], messages[j+1] = messages[j+1], messages[j]
			}
		}
	}
	return messages
}

// FilterMessages filters messages based on criteria
func (sm *StateManager) FilterMessages(
	messages []contextpkg.Message,
	criteria MessageFilterCriteria,
) []contextpkg.Message {
	filtered := make([]contextpkg.Message, 0)

	for _, msg := range messages {
		if sm.matchesMessageCriteria(msg, criteria) {
			filtered = append(filtered, msg)
		}
	}

	return filtered
}

// MessageFilterCriteria defines criteria for filtering messages
type MessageFilterCriteria struct {
	Role      string        `json:"role,omitempty"`
	AgentName string        `json:"agent_name,omitempty"`
	MaxAge    time.Duration `json:"max_age,omitempty"`
	MinLength int           `json:"min_length,omitempty"`
	MaxLength int           `json:"max_length,omitempty"`
	Contains  string        `json:"contains,omitempty"`
}

// matchesMessageCriteria checks if a message matches the filter criteria
func (sm *StateManager) matchesMessageCriteria(
	msg contextpkg.Message,
	criteria MessageFilterCriteria,
) bool {
	// Check role
	if criteria.Role != "" && msg.Role != criteria.Role {
		return false
	}

	// Check agent name
	if criteria.AgentName != "" && msg.Name != criteria.AgentName {
		return false
	}

	// Check age
	if criteria.MaxAge > 0 {
		age := time.Since(msg.Timestamp)
		if age > criteria.MaxAge {
			return false
		}
	}

	// Check length
	contentLength := len(msg.Content)
	if criteria.MinLength > 0 && contentLength < criteria.MinLength {
		return false
	}
	if criteria.MaxLength > 0 && contentLength > criteria.MaxLength {
		return false
	}

	// Check content contains
	if criteria.Contains != "" && !strings.Contains(msg.Content, criteria.Contains) {
		return false
	}

	return true
}

// TrimMessages trims messages to fit within token limit
func (sm *StateManager) TrimMessages(
	messages []contextpkg.Message,
	maxTokens int,
) []contextpkg.Message {
	if len(messages) == 0 {
		return messages
	}

	// Start from the end (most recent) and work backwards
	trimmed := make([]contextpkg.Message, 0)
	currentTokens := 0

	for i := len(messages) - 1; i >= 0; i-- {
		msgTokens := sm.estimateTokens(messages[i].Content)
		if currentTokens+msgTokens <= maxTokens {
			trimmed = append([]contextpkg.Message{messages[i]}, trimmed...)
			currentTokens += msgTokens
		} else {
			break
		}
	}

	return trimmed
}
