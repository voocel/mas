package mas

import (
	"context"
	"sync"
	"time"
)

// Memory represents the memory system for agents
type Memory interface {
	// Add adds a message to memory
	Add(ctx context.Context, role, content string) error

	// GetHistory retrieves recent conversation history
	GetHistory(ctx context.Context, limit int) ([]Message, error)

	// Clear clears all memory
	Clear() error

	// Count returns the number of messages in memory
	Count() int
}

// Message represents a single message in memory
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Role constants for messages
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// MemoryConfig contains configuration for memory systems
type MemoryConfig struct {
	MaxMessages int                    `json:"max_messages"`
	TTL         time.Duration          `json:"ttl,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// DefaultMemoryConfig returns a default memory configuration
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxMessages: 100,
		Metadata:    make(map[string]interface{}),
	}
}

// ConversationMemory implements an in-memory conversation memory
type ConversationMemory struct {
	messages []Message
	config   MemoryConfig
	mu       sync.RWMutex
}

// NewConversationMemory creates a new conversation memory with default config
func NewConversationMemory(maxMessages int) Memory {
	config := DefaultMemoryConfig()
	config.MaxMessages = maxMessages
	return NewConversationMemoryWithConfig(config)
}

// NewConversationMemoryWithConfig creates a new conversation memory with custom config
func NewConversationMemoryWithConfig(config MemoryConfig) Memory {
	return &ConversationMemory{
		messages: make([]Message, 0),
		config:   config,
	}
}

// Add adds a message to the conversation memory
func (m *ConversationMemory) Add(ctx context.Context, role, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	message := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	m.messages = append(m.messages, message)

	// Trim messages if we exceed the limit
	if len(m.messages) > m.config.MaxMessages {
		// Keep the most recent messages
		m.messages = m.messages[len(m.messages)-m.config.MaxMessages:]
	}

	return nil
}

// GetHistory retrieves recent conversation history
func (m *ConversationMemory) GetHistory(ctx context.Context, limit int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.messages) {
		limit = len(m.messages)
	}

	// Return the most recent messages
	start := len(m.messages) - limit
	if start < 0 {
		start = 0
	}

	// Create a copy to avoid race conditions
	result := make([]Message, limit)
	copy(result, m.messages[start:])

	return result, nil
}

// Clear clears all messages from memory
func (m *ConversationMemory) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]Message, 0)
	return nil
}

// Count returns the number of messages in memory
func (m *ConversationMemory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.messages)
}

// GetAllMessages returns all messages (for debugging/testing)
func (m *ConversationMemory) GetAllMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// AddWithMetadata adds a message with custom metadata
func (m *ConversationMemory) AddWithMetadata(ctx context.Context, role, content string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	message := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  metadata,
	}

	m.messages = append(m.messages, message)

	// Trim messages if we exceed the limit
	if len(m.messages) > m.config.MaxMessages {
		m.messages = m.messages[len(m.messages)-m.config.MaxMessages:]
	}

	return nil
}

// FilterMessages returns messages that match the given filter function
func (m *ConversationMemory) FilterMessages(filter func(Message) bool) []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Message
	for _, msg := range m.messages {
		if filter(msg) {
			result = append(result, msg)
		}
	}

	return result
}

// GetMessagesByRole returns all messages from a specific role
func (m *ConversationMemory) GetMessagesByRole(role string) []Message {
	return m.FilterMessages(func(msg Message) bool {
		return msg.Role == role
	})
}

// GetMessagesAfter returns messages after a specific timestamp
func (m *ConversationMemory) GetMessagesAfter(timestamp time.Time) []Message {
	return m.FilterMessages(func(msg Message) bool {
		return msg.Timestamp.After(timestamp)
	})
}

// SummaryMemory implements a memory that maintains summaries of conversations
type SummaryMemory struct {
	summary   string
	recent    []Message
	config    MemoryConfig
	mu        sync.RWMutex
}

// NewSummaryMemory creates a new summary-based memory
func NewSummaryMemory(maxRecentMessages int) Memory {
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