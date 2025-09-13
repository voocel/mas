package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

// InMemoryStore implements a simple in-memory memory store
type InMemoryStore struct {
	memories map[string]*contextpkg.Memory
	mutex    sync.RWMutex
	maxSize  int
}

// NewInMemoryStore creates a new in-memory memory store
func NewInMemoryStore(maxSize int) *InMemoryStore {
	if maxSize <= 0 {
		maxSize = 1000 // Default size
	}

	return &InMemoryStore{
		memories: make(map[string]*contextpkg.Memory),
		maxSize:  maxSize,
	}
}

// Store stores a memory in the store
func (ims *InMemoryStore) Store(ctx context.Context, memory *contextpkg.Memory) error {
	if memory == nil {
		return fmt.Errorf("memory cannot be nil")
	}

	if memory.ID == "" {
		memory.ID = generateMemoryID()
	}

	ims.mutex.Lock()
	defer ims.mutex.Unlock()

	// Check if we need to evict old memories
	if len(ims.memories) >= ims.maxSize {
		ims.evictOldMemories()
	}

	ims.memories[memory.ID] = memory
	return nil
}

// Retrieve retrieves memories based on criteria
func (ims *InMemoryStore) Retrieve(ctx context.Context, criteria contextpkg.MemoryCriteria) ([]*contextpkg.Memory, error) {
	ims.mutex.RLock()
	defer ims.mutex.RUnlock()

	var results []*contextpkg.Memory

	for _, memory := range ims.memories {
		if ims.matchesCriteria(memory, criteria) {
			results = append(results, memory)
		}
	}

	// Sort by importance and recency
	sort.Slice(results, func(i, j int) bool {
		if results[i].Importance != results[j].Importance {
			return results[i].Importance > results[j].Importance
		}
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Apply limit
	if criteria.Limit > 0 && len(results) > criteria.Limit {
		results = results[:criteria.Limit]
	}

	return results, nil
}

// Delete deletes a memory by ID
func (ims *InMemoryStore) Delete(ctx context.Context, id string) error {
	ims.mutex.Lock()
	defer ims.mutex.Unlock()

	if _, exists := ims.memories[id]; !exists {
		return fmt.Errorf("memory with ID %s not found", id)
	}

	delete(ims.memories, id)
	return nil
}

// Search searches for memories using text matching
func (ims *InMemoryStore) Search(ctx context.Context, query string, limit int) ([]*contextpkg.Memory, error) {
	ims.mutex.RLock()
	defer ims.mutex.RUnlock()

	if query == "" {
		return []*contextpkg.Memory{}, nil
	}

	queryLower := strings.ToLower(query)
	var results []*contextpkg.Memory

	for _, memory := range ims.memories {
		score := ims.calculateRelevanceScore(memory, queryLower)
		if score > 0 {
			// Create a copy with the score stored in a temporary field
			memoryCopy := *memory
			memoryCopy.Importance = score // Use importance field to store relevance score
			results = append(results, &memoryCopy)
		}
	}

	// Sort by relevance score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Importance > results[j].Importance
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// matchesCriteria checks if a memory matches the given criteria
func (ims *InMemoryStore) matchesCriteria(memory *contextpkg.Memory, criteria contextpkg.MemoryCriteria) bool {
	// Check type
	if criteria.Type != "" && memory.Type != criteria.Type {
		return false
	}

	// Check agent ID
	if criteria.AgentID != "" && memory.AgentID != criteria.AgentID {
		return false
	}

	// Check session ID
	if criteria.SessionID != "" && memory.SessionID != criteria.SessionID {
		return false
	}

	// Check minimum importance
	if criteria.MinImportance > 0 && memory.Importance < criteria.MinImportance {
		return false
	}

	// Check age
	if criteria.MaxAge > 0 {
		age := time.Since(memory.Timestamp)
		if age > criteria.MaxAge {
			return false
		}
	}

	// Check query
	if criteria.Query != "" {
		queryLower := strings.ToLower(criteria.Query)
		contentLower := strings.ToLower(memory.Content)
		if !strings.Contains(contentLower, queryLower) {
			return false
		}
	}

	return true
}

// calculateRelevanceScore calculates relevance score for search
func (ims *InMemoryStore) calculateRelevanceScore(memory *contextpkg.Memory, queryLower string) float64 {
	contentLower := strings.ToLower(memory.Content)

	// Simple text matching score
	if strings.Contains(contentLower, queryLower) {
		score := 0.5 // Base score for containing the query

		// Boost score based on how early the query appears
		index := strings.Index(contentLower, queryLower)
		if index == 0 {
			score += 0.3 // Starts with query
		} else if index < 50 {
			score += 0.2 // Query appears early
		} else {
			score += 0.1 // Query appears later
		}

		// Boost score based on memory importance
		score += memory.Importance * 0.3

		// Boost score based on recency
		age := time.Since(memory.Timestamp)
		if age < time.Hour {
			score += 0.2
		} else if age < 24*time.Hour {
			score += 0.1
		}

		return score
	}

	return 0
}

// evictOldMemories removes old memories to make space
func (ims *InMemoryStore) evictOldMemories() {
	// Convert to slice for sorting
	memories := make([]*contextpkg.Memory, 0, len(ims.memories))
	for _, memory := range ims.memories {
		memories = append(memories, memory)
	}

	// Sort by importance and age (least important and oldest first)
	sort.Slice(memories, func(i, j int) bool {
		if memories[i].Importance != memories[j].Importance {
			return memories[i].Importance < memories[j].Importance
		}
		return memories[i].Timestamp.Before(memories[j].Timestamp)
	})

	// Remove the least important/oldest memories
	toRemove := len(memories) - ims.maxSize + 1
	if toRemove > len(memories)/2 {
		toRemove = len(memories) / 2 // Don't remove more than half
	}

	for i := 0; i < toRemove; i++ {
		delete(ims.memories, memories[i].ID)
	}
}

// GetStats returns statistics about the memory store
func (ims *InMemoryStore) GetStats() MemoryStats {
	ims.mutex.RLock()
	defer ims.mutex.RUnlock()

	stats := MemoryStats{
		TotalMemories: len(ims.memories),
		MaxSize:       ims.maxSize,
		TypeCounts:    make(map[contextpkg.MemoryType]int),
		AgentCounts:   make(map[string]int),
	}

	for _, memory := range ims.memories {
		stats.TypeCounts[memory.Type]++
		stats.AgentCounts[memory.AgentID]++
	}

	return stats
}

// MemoryStats represents statistics about the memory store
type MemoryStats struct {
	TotalMemories int                           `json:"total_memories"`
	MaxSize       int                           `json:"max_size"`
	TypeCounts    map[contextpkg.MemoryType]int `json:"type_counts"`
	AgentCounts   map[string]int                `json:"agent_counts"`
}

// Clear clears all memories from the store
func (ims *InMemoryStore) Clear() error {
	ims.mutex.Lock()
	defer ims.mutex.Unlock()

	ims.memories = make(map[string]*contextpkg.Memory)
	return nil
}

// GetByID retrieves a memory by ID
func (ims *InMemoryStore) GetByID(ctx context.Context, id string) (*contextpkg.Memory, error) {
	ims.mutex.RLock()
	defer ims.mutex.RUnlock()

	memory, exists := ims.memories[id]
	if !exists {
		return nil, fmt.Errorf("memory with ID %s not found", id)
	}

	// Update access count and last access time
	memory.AccessCount++
	memory.LastAccess = time.Now()

	return memory, nil
}

// UpdateImportance updates the importance of a memory
func (ims *InMemoryStore) UpdateImportance(ctx context.Context, id string, importance float64) error {
	ims.mutex.Lock()
	defer ims.mutex.Unlock()

	memory, exists := ims.memories[id]
	if !exists {
		return fmt.Errorf("memory with ID %s not found", id)
	}

	memory.Importance = importance
	return nil
}

// GetMemoriesByType retrieves all memories of a specific type
func (ims *InMemoryStore) GetMemoriesByType(ctx context.Context, memoryType contextpkg.MemoryType) ([]*contextpkg.Memory, error) {
	criteria := contextpkg.MemoryCriteria{
		Type: memoryType,
	}
	return ims.Retrieve(ctx, criteria)
}

// GetMemoriesByAgent retrieves all memories for a specific agent
func (ims *InMemoryStore) GetMemoriesByAgent(ctx context.Context, agentID string) ([]*contextpkg.Memory, error) {
	criteria := contextpkg.MemoryCriteria{
		AgentID: agentID,
	}
	return ims.Retrieve(ctx, criteria)
}

// generateMemoryID generates a unique memory ID
func generateMemoryID() string {
	return fmt.Sprintf("memory_%d", time.Now().UnixNano())
}

// InMemoryVectorStore implements a simple in-memory vector store
type InMemoryVectorStore struct {
	vectors map[string]VectorEntry
	mutex   sync.RWMutex
}

// VectorEntry represents a vector entry
type VectorEntry struct {
	ID       string                 `json:"id"`
	Vector   []float64              `json:"vector"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// NewInMemoryVectorStore creates a new in-memory vector store
func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		vectors: make(map[string]VectorEntry),
	}
}

// Store stores a vector in the store
func (ivs *InMemoryVectorStore) Store(ctx context.Context, id string, vector []float64, metadata map[string]interface{}) error {
	ivs.mutex.Lock()
	defer ivs.mutex.Unlock()

	content := ""
	if c, ok := metadata["content"].(string); ok {
		content = c
	}

	ivs.vectors[id] = VectorEntry{
		ID:       id,
		Vector:   vector,
		Content:  content,
		Metadata: metadata,
	}

	return nil
}

// Search searches for similar vectors (simplified implementation)
func (ivs *InMemoryVectorStore) Search(ctx context.Context, query string, limit int, category string) ([]contextpkg.VectorSearchResult, error) {
	ivs.mutex.RLock()
	defer ivs.mutex.RUnlock()

	var results []contextpkg.VectorSearchResult

	// Simple text-based search (in a real implementation, you'd use vector similarity)
	queryLower := strings.ToLower(query)
	for _, entry := range ivs.vectors {
		// Filter by category if specified
		if category != "" {
			if cat, ok := entry.Metadata["category"].(string); !ok || cat != category {
				continue
			}
		}

		// Simple text matching
		contentLower := strings.ToLower(entry.Content)
		if strings.Contains(contentLower, queryLower) {
			score := 0.8 // Simplified scoring
			results = append(results, contextpkg.VectorSearchResult{
				ID:       entry.ID,
				Score:    score,
				Content:  entry.Content,
				Metadata: entry.Metadata,
			})
		}
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Delete deletes a vector by ID
func (ivs *InMemoryVectorStore) Delete(ctx context.Context, id string) error {
	ivs.mutex.Lock()
	defer ivs.mutex.Unlock()

	delete(ivs.vectors, id)
	return nil
}
