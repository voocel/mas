package memory

import (
	"context"
	"sync"
	"time"
)

// Message represents a single message in memory
type Message struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Memory represents the memory system for agents
type Memory interface {
	Add(ctx context.Context, role, content string) error
	GetHistory(ctx context.Context, limit int) ([]Message, error)
	Clear() error
	Count() int
}

// MemoryConfig contains configuration for memory systems
type MemoryConfig struct {
	MaxMessages int                    `json:"max_messages"`
	TTL         time.Duration          `json:"ttl,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ConversationMemory implements an in-memory conversation memory
type ConversationMemory struct {
	messages []Message
	config   MemoryConfig
	mu       sync.RWMutex
}

// NewConversation creates a new conversation memory with default config
func NewConversation(maxMessages int) Memory {
	config := DefaultMemoryConfig()
	config.MaxMessages = maxMessages
	return NewConversationWithConfig(config)
}

// NewConversationWithConfig creates a new conversation memory with custom config
func NewConversationWithConfig(config MemoryConfig) Memory {
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

// DefaultMemoryConfig returns a default memory configuration
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxMessages: 100,
		Metadata:    make(map[string]interface{}),
	}
}