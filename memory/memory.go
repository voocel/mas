package memory

import (
	"context"
	"time"

	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/schema"
)

// Memory defines the core interface for the memory system.
type Memory interface {
	// Add adds a message to memory.
	Add(ctx context.Context, message schema.Message) error

	// Query queries for relevant memory content.
	Query(ctx context.Context, query string, limit int) ([]MemoryItem, error)

	// Clear clears the memory.
	Clear(ctx context.Context) error

	// GetHistory gets the full conversation history.
	GetHistory(ctx context.Context) ([]schema.Message, error)

	// GetRecentHistory gets the N most recent history records.
	GetRecentHistory(ctx context.Context, limit int) ([]schema.Message, error)
}

type MemoryItem struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	Score     float64                `json:"score,omitempty"` // Similarity score, used for retrieval.
}

// MemoryConfig is the configuration for memory.
type MemoryConfig struct {
	// MaxHistory is the maximum number of history records.
	MaxHistory int `json:"max_history"`

	// RetentionDays is the number of days to retain records, 0 means forever.
	RetentionDays int `json:"retention_days"`

	// EnableSearch enables semantic search.
	EnableSearch bool `json:"enable_search"`

	// SummaryModel is the LLM model used for AI summarization (required, Summarize will return an error if nil).
	SummaryModel llm.ChatModel `json:"-"`
}

// DefaultMemoryConfig is the default memory configuration.
var DefaultMemoryConfig = &MemoryConfig{
	MaxHistory:    100,
	RetentionDays: 30,
	EnableSearch:  false,
	SummaryModel:  nil,
}

// ConversationMemory is the interface for conversation memory.
// It is specifically used to manage the conversation history of an agent.
type ConversationMemory interface {
	Memory

	// AddConversationTurn adds a turn of conversation (user message + assistant reply).
	AddConversationTurn(ctx context.Context, userMsg, assistantMsg schema.Message) error

	// GetConversationContext gets the conversation context for LLM calls.
	GetConversationContext(ctx context.Context) ([]schema.Message, error)

	// Summarize summarizes the conversation history using AI.
	// If model is nil, the configured SummaryModel is used.
	// If model is not nil, the specified model is used.
	Summarize(ctx context.Context, model ...llm.ChatModel) (string, error)
}

// SharedMemory is the interface for shared memory.
// It is used to share information among multiple agents.
type SharedMemory interface {
	Memory

	// AddWithScope adds memory with a scope.
	AddWithScope(ctx context.Context, scope string, message schema.Message) error

	// QueryByScope queries by scope.
	QueryByScope(ctx context.Context, scope string, query string, limit int) ([]MemoryItem, error)

	// GetScopes gets all scopes.
	GetScopes(ctx context.Context) ([]string, error)
}

// MemoryManager is the memory manager.
// It uniformly manages different types of memory.
type MemoryManager struct {
	conversation ConversationMemory
	shared       SharedMemory
	config       *MemoryConfig
}

// NewMemoryManager creates a new memory manager.
func NewMemoryManager(config *MemoryConfig) *MemoryManager {
	if config == nil {
		config = DefaultMemoryConfig
	}

	return &MemoryManager{
		conversation: NewConversationMemory(config),
		shared:       NewSharedMemory(config),
		config:       config,
	}
}

// Conversation gets the conversation memory.
func (m *MemoryManager) Conversation() ConversationMemory {
	return m.conversation
}

// Shared gets the shared memory.
func (m *MemoryManager) Shared() SharedMemory {
	return m.shared
}

// SetConversationMemory sets the conversation memory implementation.
func (m *MemoryManager) SetConversationMemory(memory ConversationMemory) {
	m.conversation = memory
}

// SetSharedMemory sets the shared memory implementation.
func (m *MemoryManager) SetSharedMemory(memory SharedMemory) {
	m.shared = memory
}

// MemoryProvider is the interface for a memory provider (for extending to different storage backends).
type MemoryProvider interface {
	Store(ctx context.Context, item MemoryItem) error

	Retrieve(ctx context.Context, query string, limit int) ([]MemoryItem, error)

	Delete(ctx context.Context, id string) error

	List(ctx context.Context, limit int, offset int) ([]MemoryItem, error)

	Close() error
}

// MemoryStats is the statistics for memory.
type MemoryStats struct {
	TotalItems    int       `json:"total_items"`
	TotalSize     int64     `json:"total_size"`
	OldestItem    time.Time `json:"oldest_item"`
	NewestItem    time.Time `json:"newest_item"`
	AverageScore  float64   `json:"average_score"`
	RetentionDays int       `json:"retention_days"`
	LastCleanup   time.Time `json:"last_cleanup"`
}

// GetStats gets the memory statistics.
func (m *MemoryManager) GetStats(ctx context.Context) (*MemoryStats, error) {
	// todo: implement statistics logic
	return &MemoryStats{}, nil
}

// Cleanup cleans up expired memory.
func (m *MemoryManager) Cleanup(ctx context.Context) error {
	if m.config.RetentionDays <= 0 {
		return nil // Retain forever
	}

	// Implement cleanup logic
	return nil
}
