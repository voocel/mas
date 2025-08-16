package mas

import (
	"context"
	"sync"
	"time"
)

type conversationMemory struct {
	messages []Message
	config   MemoryConfig
	mu       sync.RWMutex
}

type summaryMemory struct {
	summary string
	recent  []Message
	config  MemoryConfig
	mu      sync.RWMutex
}

func NewConversationMemory(maxMessages int) Memory {
	return &conversationMemory{
		messages: make([]Message, 0),
		config:   MemoryConfig{MaxMessages: maxMessages},
	}
}

func NewConversationMemoryWithConfig(config MemoryConfig) Memory {
	return &conversationMemory{
		messages: make([]Message, 0),
		config:   config,
	}
}

func NewSummaryMemory(maxRecentMessages int) Memory {
	return &summaryMemory{
		summary: "",
		recent:  make([]Message, 0),
		config:  MemoryConfig{MaxMessages: maxRecentMessages},
	}
}

func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		MaxMessages: 100,
		Metadata:    make(map[string]interface{}),
	}
}

func (m *conversationMemory) Add(ctx context.Context, role, content string) error {
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
		m.messages = m.messages[len(m.messages)-m.config.MaxMessages:]
	}

	return nil
}

func (m *conversationMemory) GetHistory(ctx context.Context, limit int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.messages) {
		limit = len(m.messages)
	}

	start := len(m.messages) - limit
	if start < 0 {
		start = 0
	}

	// Create a copy to avoid race conditions
	result := make([]Message, limit)
	copy(result, m.messages[start:])

	return result, nil
}

func (m *conversationMemory) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make([]Message, 0)
	return nil
}

func (m *conversationMemory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

func (m *summaryMemory) Add(ctx context.Context, role, content string) error {
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
		m.recent = m.recent[len(m.recent)-m.config.MaxMessages:]
	}

	return nil
}

func (m *summaryMemory) GetHistory(ctx context.Context, limit int) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Message

	// Add summary as a system message if it exists
	if m.summary != "" {
		result = append(result, Message{
			Role:      RoleSystem,
			Content:   "Previous conversation summary: " + m.summary,
			Timestamp: time.Now(),
			Metadata:  make(map[string]interface{}),
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

func (m *summaryMemory) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summary = ""
	m.recent = make([]Message, 0)
	return nil
}

func (m *summaryMemory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.recent)
}

func (m *summaryMemory) SetSummary(summary string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summary = summary
}

func (m *summaryMemory) GetSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.summary
}

func (m *summaryMemory) GetRecentMessages() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Message, len(m.recent))
	copy(result, m.recent)
	return result
}

type MemoryOption func(*MemoryConfig)

func WithMaxMessages(max int) MemoryOption {
	return func(c *MemoryConfig) {
		c.MaxMessages = max
	}
}

func WithTTL(ttl time.Duration) MemoryOption {
	return func(c *MemoryConfig) {
		c.TTL = ttl
	}
}

func WithMetadata(metadata map[string]interface{}) MemoryOption {
	return func(c *MemoryConfig) {
		c.Metadata = metadata
	}
}

func NewChatMemory() Memory {
	return NewConversationMemory(50)
}

func NewShortTermMemory() Memory {
	config := MemoryConfig{
		MaxMessages: 20,
		TTL:         30 * time.Minute,
		Metadata:    make(map[string]interface{}),
	}
	return NewConversationMemoryWithConfig(config)
}

func NewLongTermMemory() Memory {
	return NewConversationMemory(200)
}
