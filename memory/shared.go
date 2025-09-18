package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/voocel/mas/schema"
)

// sharedMemory is the in-memory implementation of shared memory.
// It is used to share information and knowledge among multiple agents.
type sharedMemory struct {
	items  map[string][]MemoryItem // scope -> items
	config *MemoryConfig
	mutex  sync.RWMutex
}

// NewSharedMemory creates a new shared memory.
func NewSharedMemory(config *MemoryConfig) SharedMemory {
	if config == nil {
		config = DefaultMemoryConfig
	}

	return &sharedMemory{
		items:  make(map[string][]MemoryItem),
		config: config,
	}
}

// Add adds a message to the shared memory (default scope).
func (s *sharedMemory) Add(ctx context.Context, message schema.Message) error {
	return s.AddWithScope(ctx, "default", message)
}

// AddWithScope adds memory with a scope.
func (s *sharedMemory) AddWithScope(ctx context.Context, scope string, message schema.Message) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	item := MemoryItem{
		ID:        generateID(),
		Content:   message.Content,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"role":       message.Role,
			"scope":      scope,
			"agent_id":   message.Metadata["agent_id"],
			"message_id": message.ID,
		},
	}

	// Copy the message's metadata.
	if message.Metadata != nil {
		for k, v := range message.Metadata {
			item.Metadata[k] = v
		}
	}

	// Add to the specified scope.
	if s.items[scope] == nil {
		s.items[scope] = make([]MemoryItem, 0)
	}

	s.items[scope] = append(s.items[scope], item)

	// Limit the number of items per scope.
	if len(s.items[scope]) > s.config.MaxHistory {
		// Keep the most recent items.
		keepCount := s.config.MaxHistory
		s.items[scope] = s.items[scope][len(s.items[scope])-keepCount:]
	}

	return nil
}

// Query queries for relevant memory content (all scopes).
func (s *sharedMemory) Query(ctx context.Context, query string, limit int) ([]MemoryItem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	results := make([]MemoryItem, 0)

	// Search all scopes.
	for scope := range s.items {
		scopeResults, err := s.queryScope(scope, query, limit-len(results))
		if err != nil {
			continue // Ignore errors in individual scopes.
		}
		results = append(results, scopeResults...)

		if len(results) >= limit {
			break
		}
	}

	// Sort by timestamp (newest first).
	sortMemoryItemsByTime(results)

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// QueryByScope queries by scope.
func (s *sharedMemory) QueryByScope(ctx context.Context, scope string, query string, limit int) ([]MemoryItem, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.queryScope(scope, query, limit)
}

// GetHistory gets the full shared memory history.
func (s *sharedMemory) GetHistory(ctx context.Context) ([]schema.Message, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	messages := make([]schema.Message, 0)

	// Collect messages from all scopes.
	for _, items := range s.items {
		for _, item := range items {
			msg := schema.Message{
				ID:        item.Metadata["message_id"].(string),
				Content:   item.Content,
				Timestamp: item.Timestamp,
				Metadata:  item.Metadata,
			}

			// Restore role information.
			if role, ok := item.Metadata["role"].(schema.Role); ok {
				msg.Role = role
			}

			messages = append(messages, msg)
		}
	}

	sortMessagesByTime(messages)

	return messages, nil
}

// GetRecentHistory gets the N most recent history records.
func (s *sharedMemory) GetRecentHistory(ctx context.Context, limit int) ([]schema.Message, error) {
	history, err := s.GetHistory(ctx)
	if err != nil {
		return nil, err
	}

	if len(history) <= limit {
		return history, nil
	}

	// Return the most recent records.
	return history[len(history)-limit:], nil
}

// GetScopes gets all scopes.
func (s *sharedMemory) GetScopes(ctx context.Context) ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	scopes := make([]string, 0, len(s.items))
	for scope := range s.items {
		scopes = append(scopes, scope)
	}

	return scopes, nil
}

// Clear clears all shared memory.
func (s *sharedMemory) Clear(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.items = make(map[string][]MemoryItem)
	return nil
}

// ClearScope clears the memory of a specified scope.
func (s *sharedMemory) ClearScope(ctx context.Context, scope string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.items, scope)
	return nil
}

// queryScope queries a specified scope.
func (s *sharedMemory) queryScope(scope string, query string, limit int) ([]MemoryItem, error) {
	items, exists := s.items[scope]
	if !exists {
		return []MemoryItem{}, nil
	}

	if !s.config.EnableSearch {
		return s.getRecentItemsFromScope(scope, limit), nil
	}

	// Simple text matching search.
	results := make([]MemoryItem, 0)
	for i := len(items) - 1; i >= 0 && len(results) < limit; i-- {
		item := items[i]
		if containsIgnoreCase(item.Content, query) {
			// Create a copy and set the score.
			result := item
			result.Score = 1.0 // Simple match score.
			results = append(results, result)
		}
	}

	return results, nil
}

// getRecentItemsFromScope gets the most recent items from a specified scope.
func (s *sharedMemory) getRecentItemsFromScope(scope string, limit int) []MemoryItem {
	items, exists := s.items[scope]
	if !exists {
		return []MemoryItem{}
	}

	start := len(items) - limit
	if start < 0 {
		start = 0
	}

	result := make([]MemoryItem, len(items)-start)
	copy(result, items[start:])
	return result
}

func generateID() string {
	return fmt.Sprintf("mem_%d", time.Now().UnixNano())
}

func sortMemoryItemsByTime(items []MemoryItem) {
	// Simple bubble sort, in descending order of timestamp.
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].Timestamp.Before(items[j].Timestamp) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func sortMessagesByTime(messages []schema.Message) {
	// Simple bubble sort, in ascending order of timestamp.
	for i := 0; i < len(messages)-1; i++ {
		for j := i + 1; j < len(messages); j++ {
			if messages[i].Timestamp.After(messages[j].Timestamp) {
				messages[i], messages[j] = messages[j], messages[i]
			}
		}
	}
}
