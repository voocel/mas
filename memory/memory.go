package memory

import (
	"context"
	"time"
)

// Memory defines the memory system for agents
type Memory interface {
	// Add adds a memory item
	Add(ctx context.Context, item MemoryItem) error

	// Get retrieves a memory item by ID
	Get(ctx context.Context, id string) (MemoryItem, error)

	// Search searches for related memories based on a query
	Search(ctx context.Context, query string, limit int) ([]MemoryItem, error)

	// GetRecent retrieves the most recent n memories
	GetRecent(ctx context.Context, n int) ([]MemoryItem, error)

	// Clear clears all memories
	Clear(ctx context.Context) error
}

// MemoryType defines memory types
type MemoryType string

const (
	// TypeObservation observation memory
	TypeObservation MemoryType = "observation"
	// TypeThought thought memory
	TypeThought MemoryType = "thought"
	// TypeAction action memory
	TypeAction MemoryType = "action"
	// TypeResult result memory
	TypeResult MemoryType = "result"
)

type MemoryItem struct {
	ID        string                 `json:"id"`
	Content   interface{}            `json:"content"`
	Type      MemoryType             `json:"type"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Config struct {
	Type        string
	Capacity    int
	Persistence bool
	StoragePath string
}

func New(config Config) Memory {
	switch config.Type {
	case "inmemory":
		return NewInMemory(config)
	case "vectorstore":
		return NewVectorStore(config)
	default:
		return NewInMemory(config)
	}
}

type InMemory struct {
	items    []MemoryItem
	capacity int
}

func NewInMemory(config Config) *InMemory {
	capacity := 1000
	if config.Capacity > 0 {
		capacity = config.Capacity
	}

	return &InMemory{
		items:    make([]MemoryItem, 0),
		capacity: capacity,
	}
}

func (m *InMemory) Add(ctx context.Context, item MemoryItem) error {
	if len(m.items) >= m.capacity {
		// remove the oldest memory
		m.items = m.items[1:]
	}
	m.items = append(m.items, item)
	return nil
}

func (m *InMemory) Get(ctx context.Context, id string) (MemoryItem, error) {
	for _, item := range m.items {
		if item.ID == id {
			return item, nil
		}
	}
	return MemoryItem{}, ErrMemoryNotFound
}

func (m *InMemory) Search(ctx context.Context, query string, limit int) ([]MemoryItem, error) {
	// TODO: implement fuzzy matching, vector search, etc.
	result := make([]MemoryItem, 0)
	count := 0

	for i := len(m.items) - 1; i >= 0 && count < limit; i-- {
		item := m.items[i]
		if content, ok := item.Content.(string); ok && contains(content, query) {
			result = append(result, item)
			count++
		}
	}

	return result, nil
}

func (m *InMemory) GetRecent(ctx context.Context, n int) ([]MemoryItem, error) {
	if n <= 0 || len(m.items) == 0 {
		return []MemoryItem{}, nil
	}

	if n >= len(m.items) {
		return m.items, nil
	}

	return m.items[len(m.items)-n:], nil
}

func (m *InMemory) Clear(ctx context.Context) error {
	m.items = make([]MemoryItem, 0)
	return nil
}

	// VectorStore implements a memory system based on vector storage
type VectorStore struct {
	items []MemoryItem
	// TODO: implement vector CLI
}

func NewVectorStore(config Config) *VectorStore {
	// TODO: connect to Milvus
	return &VectorStore{
		items: make([]MemoryItem, 0),
	}
}

func (v *VectorStore) Add(ctx context.Context, item MemoryItem) error {
	v.items = append(v.items, item)
	return nil
}

func (v *VectorStore) Get(ctx context.Context, id string) (MemoryItem, error) {
	for _, item := range v.items {
		if item.ID == id {
			return item, nil
		}
	}
	return MemoryItem{}, ErrMemoryNotFound
}

func (v *VectorStore) Search(ctx context.Context, query string, limit int) ([]MemoryItem, error) {
	// TODO: perform vector similarity search
	result := make([]MemoryItem, 0)
	count := 0

	for i := len(v.items) - 1; i >= 0 && count < limit; i-- {
		item := v.items[i]
		if content, ok := item.Content.(string); ok && contains(content, query) {
			result = append(result, item)
			count++
		}
	}

	return result, nil
}

func (v *VectorStore) GetRecent(ctx context.Context, n int) ([]MemoryItem, error) {
	if n <= 0 || len(v.items) == 0 {
		return []MemoryItem{}, nil
	}

	if n >= len(v.items) {
		return v.items, nil
	}

	return v.items[len(v.items)-n:], nil
}

func (v *VectorStore) Clear(ctx context.Context) error {
	v.items = make([]MemoryItem, 0)
	return nil
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr
}

var (
	ErrMemoryNotFound = NewMemoryError("memory item not found")
)

type MemoryError struct {
	msg string
}

func NewMemoryError(msg string) *MemoryError {
	return &MemoryError{msg: msg}
}

func (e *MemoryError) Error() string {
	return e.msg
}
