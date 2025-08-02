package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/voocel/mas"
)

// Conversation provides helper functions for conversation memory
func Conversation(maxMessages int) mas.Memory {
	return mas.NewConversationMemory(maxMessages)
}

// ConversationWithConfig creates conversation memory with custom configuration
func ConversationWithConfig(config mas.MemoryConfig) mas.Memory {
	return mas.NewConversationMemoryWithConfig(config)
}

// Summary provides helper function for summary memory
func Summary(maxRecentMessages int) mas.Memory {
	return mas.NewSummaryMemory(maxRecentMessages)
}

// Persistent creates a conversation memory that persists to disk
func Persistent(maxMessages int, filePath string) mas.Memory {
	return &PersistentMemory{
		inner:    mas.NewConversationMemory(maxMessages),
		filePath: filePath,
		mu:       sync.RWMutex{},
	}
}

// PersistentMemory wraps conversation memory with disk persistence
type PersistentMemory struct {
	inner    mas.Memory
	filePath string
	mu       sync.RWMutex
	loaded   bool
}

// Add adds a message to persistent memory
func (p *PersistentMemory) Add(ctx context.Context, role, content string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Load from disk if not already loaded
	if !p.loaded {
		p.loadFromDisk()
		p.loaded = true
	}

	err := p.inner.Add(ctx, role, content)
	if err != nil {
		return err
	}

	// Save to disk
	return p.saveToDisk(ctx)
}

// GetHistory retrieves conversation history from persistent memory
func (p *PersistentMemory) GetHistory(ctx context.Context, limit int) ([]mas.Message, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Load from disk if not already loaded
	if !p.loaded {
		p.mu.RUnlock()
		p.mu.Lock()
		if !p.loaded {
			p.loadFromDisk()
			p.loaded = true
		}
		p.mu.Unlock()
		p.mu.RLock()
	}

	return p.inner.GetHistory(ctx, limit)
}

// Clear clears persistent memory and removes the file
func (p *PersistentMemory) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	err := p.inner.Clear()
	if err != nil {
		return err
	}

	// Remove the file
	if _, err := os.Stat(p.filePath); err == nil {
		return os.Remove(p.filePath)
	}

	return nil
}

// Count returns the number of messages in persistent memory
func (p *PersistentMemory) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Load from disk if not already loaded
	if !p.loaded {
		p.mu.RUnlock()
		p.mu.Lock()
		if !p.loaded {
			p.loadFromDisk()
			p.loaded = true
		}
		p.mu.Unlock()
		p.mu.RLock()
	}

	return p.inner.Count()
}

// loadFromDisk loads conversation history from disk
func (p *PersistentMemory) loadFromDisk() {
	if _, err := os.Stat(p.filePath); os.IsNotExist(err) {
		return // File doesn't exist, start fresh
	}

	data, err := os.ReadFile(p.filePath)
	if err != nil {
		return // Failed to read, start fresh
	}

	var messages []mas.Message
	err = json.Unmarshal(data, &messages)
	if err != nil {
		return // Failed to parse, start fresh
	}

	// Add messages to inner memory
	ctx := context.Background()
	for _, msg := range messages {
		p.inner.Add(ctx, msg.Role, msg.Content)
	}
}

// saveToDisk saves conversation history to disk
func (p *PersistentMemory) saveToDisk(ctx context.Context) error {
	// Ensure directory exists
	dir := filepath.Dir(p.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Get all messages
	messages, err := p.inner.GetHistory(ctx, -1) // Get all messages
	if err != nil {
		return err
	}

	// Convert to JSON
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(p.filePath, data, 0644)
}

// SharedMemory provides a memory that can be shared between multiple agents
type SharedMemory struct {
	inner mas.Memory
	mu    sync.RWMutex
}

// NewSharedMemory creates a new shared memory instance
func NewSharedMemory(baseMemory mas.Memory) *SharedMemory {
	return &SharedMemory{
		inner: baseMemory,
		mu:    sync.RWMutex{},
	}
}

// Add adds a message to shared memory with thread safety
func (s *SharedMemory) Add(ctx context.Context, role, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Add(ctx, role, content)
}

// GetHistory retrieves history from shared memory with thread safety
func (s *SharedMemory) GetHistory(ctx context.Context, limit int) ([]mas.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inner.GetHistory(ctx, limit)
}

// Clear clears shared memory with thread safety
func (s *SharedMemory) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.inner.Clear()
}

// Count returns message count from shared memory with thread safety
func (s *SharedMemory) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.inner.Count()
}

// BufferedMemory provides a memory with automatic batching for performance
type BufferedMemory struct {
	inner      mas.Memory
	buffer     []mas.Message
	bufferSize int
	flushTimer *time.Timer
	mu         sync.Mutex
}

// NewBufferedMemory creates a new buffered memory instance
func NewBufferedMemory(baseMemory mas.Memory, bufferSize int, flushInterval time.Duration) *BufferedMemory {
	bm := &BufferedMemory{
		inner:      baseMemory,
		buffer:     make([]mas.Message, 0),
		bufferSize: bufferSize,
		mu:         sync.Mutex{},
	}

	// Start flush timer
	bm.flushTimer = time.AfterFunc(flushInterval, bm.flush)

	return bm
}

// Add adds a message to buffered memory
func (b *BufferedMemory) Add(ctx context.Context, role, content string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	message := mas.Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	b.buffer = append(b.buffer, message)

	// Flush if buffer is full
	if len(b.buffer) >= b.bufferSize {
		return b.flushBuffer(ctx)
	}

	return nil
}

// GetHistory retrieves history, flushing buffer first
func (b *BufferedMemory) GetHistory(ctx context.Context, limit int) ([]mas.Message, error) {
	b.mu.Lock()
	err := b.flushBuffer(ctx)
	b.mu.Unlock()

	if err != nil {
		return nil, err
	}

	return b.inner.GetHistory(ctx, limit)
}

// Clear clears both buffer and inner memory
func (b *BufferedMemory) Clear() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.buffer = make([]mas.Message, 0)
	return b.inner.Clear()
}

// Count returns total count including buffered messages
func (b *BufferedMemory) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.inner.Count() + len(b.buffer)
}

// flush is called by the timer to flush the buffer
func (b *BufferedMemory) flush() {
	b.mu.Lock()
	defer b.mu.Unlock()

	ctx := context.Background()
	b.flushBuffer(ctx)

	// Reset timer
	if b.flushTimer != nil {
		b.flushTimer.Reset(time.Minute) // Default to 1 minute
	}
}

// flushBuffer flushes all buffered messages to inner memory
func (b *BufferedMemory) flushBuffer(ctx context.Context) error {
	if len(b.buffer) == 0 {
		return nil
	}

	for _, msg := range b.buffer {
		err := b.inner.Add(ctx, msg.Role, msg.Content)
		if err != nil {
			return err
		}
	}

	b.buffer = make([]mas.Message, 0)
	return nil
}

// Close properly closes the buffered memory
func (b *BufferedMemory) Close() error {
	if b.flushTimer != nil {
		b.flushTimer.Stop()
	}

	ctx := context.Background()
	return b.flushBuffer(ctx)
}

// Helper functions for common memory patterns

// ThreadSafe wraps any memory implementation with thread safety
func ThreadSafe(baseMemory mas.Memory) mas.Memory {
	return NewSharedMemory(baseMemory)
}

// WithPersistence adds disk persistence to any memory implementation
func WithPersistence(baseMemory mas.Memory, filePath string) mas.Memory {
	// This is a simplified version - in practice you might want to wrap
	// the base memory more carefully
	return Persistent(100, filePath) // Default to 100 messages
}

// Buffered adds buffering to any memory implementation
func Buffered(baseMemory mas.Memory, bufferSize int, flushInterval time.Duration) *BufferedMemory {
	return NewBufferedMemory(baseMemory, bufferSize, flushInterval)
}

// MultiTier creates a multi-tier memory system with fast and slow storage
func MultiTier(fastMemory, slowMemory mas.Memory, fastLimit int) mas.Memory {
	return &MultiTierMemory{
		fast:      fastMemory,
		slow:      slowMemory,
		fastLimit: fastLimit,
		mu:        sync.RWMutex{},
	}
}

// MultiTierMemory implements a two-tier memory system
type MultiTierMemory struct {
	fast      mas.Memory
	slow      mas.Memory
	fastLimit int
	mu        sync.RWMutex
}

// Add adds to fast memory and manages overflow to slow memory
func (m *MultiTierMemory) Add(ctx context.Context, role, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Add to fast memory
	err := m.fast.Add(ctx, role, content)
	if err != nil {
		return err
	}

	// Check if we need to move old messages to slow memory
	if m.fast.Count() > m.fastLimit {
		// Get oldest messages from fast memory
		oldMessages, err := m.fast.GetHistory(ctx, m.fast.Count()-m.fastLimit)
		if err != nil {
			return err
		}

		// Move to slow memory
		for _, msg := range oldMessages {
			err := m.slow.Add(ctx, msg.Role, msg.Content)
			if err != nil {
				return err
			}
		}

		// Clear fast memory and re-add recent messages
		recentMessages, err := m.fast.GetHistory(ctx, m.fastLimit)
		if err != nil {
			return err
		}

		err = m.fast.Clear()
		if err != nil {
			return err
		}

		for _, msg := range recentMessages {
			err := m.fast.Add(ctx, msg.Role, msg.Content)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// GetHistory retrieves from both fast and slow memory
func (m *MultiTierMemory) GetHistory(ctx context.Context, limit int) ([]mas.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get from fast memory first
	fastMessages, err := m.fast.GetHistory(ctx, limit)
	if err != nil {
		return nil, err
	}

	// If we need more messages, get from slow memory
	if len(fastMessages) < limit {
		remaining := limit - len(fastMessages)
		slowMessages, err := m.slow.GetHistory(ctx, remaining)
		if err != nil {
			return fastMessages, nil // Return what we have from fast memory
		}

		// Combine messages (slow messages first, then fast messages)
		allMessages := make([]mas.Message, 0, len(slowMessages)+len(fastMessages))
		allMessages = append(allMessages, slowMessages...)
		allMessages = append(allMessages, fastMessages...)

		return allMessages, nil
	}

	return fastMessages, nil
}

// Clear clears both fast and slow memory
func (m *MultiTierMemory) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	err1 := m.fast.Clear()
	err2 := m.slow.Clear()

	if err1 != nil {
		return err1
	}
	return err2
}

// Count returns total count from both memories
func (m *MultiTierMemory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.fast.Count() + m.slow.Count()
}