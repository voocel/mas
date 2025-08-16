package memory

import (
	"context"
	"sync"
	"time"
)

// Role constants for messages
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// SummaryMemory implements a memory that maintains summaries of conversations
type SummaryMemory struct {
	summary   string
	recent    []Message
	config    MemoryConfig
	mu        sync.RWMutex
}

// NewSummary creates a new summary-based memory
func NewSummary(maxRecentMessages int) Memory {
	config := DefaultMemoryConfig()
	config.MaxMessages = maxRecentMessages
	
	return &SummaryMemory{
		summary: "",
		recent:  make([]Message, 0),
		config:  config,
	}
}

// Add adds a message to summary memory
func (m *SummaryMemory) Add(ctx context.Context, role, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	message := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	m.recent = append(m.recent, message)

	// When we exceed the limit, we should summarize older messages
	if len(m.recent) > m.config.MaxMessages {
		// For now, just keep recent messages
		// In a real implementation, you would use an LLM to create summaries
		m.recent = m.recent[len(m.recent)-m.config.MaxMessages:]
	}

	return nil
}

// GetHistory retrieves recent messages and summary
func (m *SummaryMemory) GetHistory(ctx context.Context, limit int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Message

	// Add summary as a system message if it exists
	if m.summary != "" {
		result = append(result, Message{
			Role:      RoleSystem,
			Content:   "Previous conversation summary: " + m.summary,
			Timestamp: time.Now(),
		})
	}

	// Add recent messages
	if limit <= 0 || limit > len(m.recent) {
		limit = len(m.recent)
	}

	start := len(m.recent) - limit
	if start < 0 {
		start = 0
	}

	recentCopy := make([]Message, limit)
	copy(recentCopy, m.recent[start:])
	result = append(result, recentCopy...)

	return result, nil
}

// Clear clears both summary and recent messages
func (m *SummaryMemory) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.summary = ""
	m.recent = make([]Message, 0)
	return nil
}

// Count returns the total count of recent messages
func (m *SummaryMemory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.recent)
}

// SetSummary updates the conversation summary
func (m *SummaryMemory) SetSummary(summary string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.summary = summary
}

// GetSummary returns the current summary
func (m *SummaryMemory) GetSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.summary
}

// GetRecentMessages returns only the recent messages (without summary)
func (m *SummaryMemory) GetRecentMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Message, len(m.recent))
	copy(result, m.recent)
	return result
}