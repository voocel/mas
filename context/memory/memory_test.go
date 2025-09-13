package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	contextpkg "github.com/voocel/mas/context"
)

func TestInMemoryStore(t *testing.T) {
	store := NewInMemoryStore(100)
	ctx := context.Background()

	// Test storing a memory
	memory := &contextpkg.Memory{
		ID:          "test_memory_1",
		Type:        contextpkg.EpisodicMemoryType,
		Content:     "This is a test memory about a conversation",
		Context:     map[string]interface{}{"agent_id": "test_agent"},
		Importance:  0.8,
		Timestamp:   time.Now(),
		AgentID:     "test_agent",
		SessionID:   "test_session",
		AccessCount: 0,
		LastAccess:  time.Now(),
	}

	err := store.Store(ctx, memory)
	if err != nil {
		t.Fatalf("Failed to store memory: %v", err)
	}

	// Test retrieving memories
	criteria := contextpkg.MemoryCriteria{
		AgentID: "test_agent",
		Limit:   10,
	}

	memories, err := store.Retrieve(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to retrieve memories: %v", err)
	}

	if len(memories) != 1 {
		t.Errorf("Expected 1 memory, got %d", len(memories))
	}

	if memories[0].ID != "test_memory_1" {
		t.Errorf("Expected memory ID 'test_memory_1', got %s", memories[0].ID)
	}

	// Test searching memories
	searchResults, err := store.Search(ctx, "conversation", 10)
	if err != nil {
		t.Fatalf("Failed to search memories: %v", err)
	}

	if len(searchResults) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(searchResults))
	}

	// Test deleting memory
	err = store.Delete(ctx, "test_memory_1")
	if err != nil {
		t.Fatalf("Failed to delete memory: %v", err)
	}

	// Verify deletion
	memories, err = store.Retrieve(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to retrieve memories after deletion: %v", err)
	}

	if len(memories) != 0 {
		t.Errorf("Expected 0 memories after deletion, got %d", len(memories))
	}
}

func TestMemoryFiltering(t *testing.T) {
	store := NewInMemoryStore(100)
	ctx := context.Background()

	// Store multiple memories
	memories := []*contextpkg.Memory{
		{
			ID:         "memory_1",
			Type:       contextpkg.EpisodicMemoryType,
			Content:    "First memory",
			Importance: 0.9,
			Timestamp:  time.Now().Add(-2 * time.Hour),
			AgentID:    "agent_1",
			SessionID:  "session_1",
		},
		{
			ID:         "memory_2",
			Type:       contextpkg.SemanticMemoryType,
			Content:    "Second memory",
			Importance: 0.7,
			Timestamp:  time.Now().Add(-1 * time.Hour),
			AgentID:    "agent_1",
			SessionID:  "session_1",
		},
		{
			ID:         "memory_3",
			Type:       contextpkg.EpisodicMemoryType,
			Content:    "Third memory",
			Importance: 0.5,
			Timestamp:  time.Now(),
			AgentID:    "agent_2",
			SessionID:  "session_2",
		},
	}

	for _, memory := range memories {
		err := store.Store(ctx, memory)
		if err != nil {
			t.Fatalf("Failed to store memory %s: %v", memory.ID, err)
		}
	}

	// Test filtering by type
	criteria := contextpkg.MemoryCriteria{
		Type:  contextpkg.EpisodicMemoryType,
		Limit: 10,
	}

	results, err := store.Retrieve(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to retrieve episodic memories: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 episodic memories, got %d", len(results))
	}

	// Test filtering by agent
	criteria = contextpkg.MemoryCriteria{
		AgentID: "agent_1",
		Limit:   10,
	}

	results, err = store.Retrieve(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to retrieve agent_1 memories: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 memories for agent_1, got %d", len(results))
	}

	// Test filtering by importance
	criteria = contextpkg.MemoryCriteria{
		MinImportance: 0.8,
		Limit:         10,
	}

	results, err = store.Retrieve(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to retrieve high importance memories: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 high importance memory, got %d", len(results))
	}

	// Test filtering by age
	criteria = contextpkg.MemoryCriteria{
		MaxAge: 30 * time.Minute,
		Limit:  10,
	}

	results, err = store.Retrieve(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to retrieve recent memories: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 recent memory, got %d", len(results))
	}
}

func TestMemoryEviction(t *testing.T) {
	store := NewInMemoryStore(3) // Small limit for testing
	ctx := context.Background()

	// Store more memories than the limit
	for i := 0; i < 5; i++ {
		memory := &contextpkg.Memory{
			ID:         fmt.Sprintf("memory_%d", i),
			Type:       contextpkg.EpisodicMemoryType,
			Content:    fmt.Sprintf("Memory content %d", i),
			Importance: float64(i) / 10.0, // Increasing importance
			Timestamp:  time.Now().Add(time.Duration(i) * time.Minute),
			AgentID:    "test_agent",
			SessionID:  "test_session",
		}

		err := store.Store(ctx, memory)
		if err != nil {
			t.Fatalf("Failed to store memory %d: %v", i, err)
		}
	}

	// Check that only the most important/recent memories are kept
	stats := store.GetStats()
	if stats.TotalMemories > 3 {
		t.Errorf("Expected at most 3 memories after eviction, got %d", stats.TotalMemories)
	}

	// Verify that the most important memories are kept
	criteria := contextpkg.MemoryCriteria{
		Limit: 10,
	}

	memories, err := store.Retrieve(ctx, criteria)
	if err != nil {
		t.Fatalf("Failed to retrieve memories after eviction: %v", err)
	}

	// The memories should be sorted by importance (highest first)
	if len(memories) > 1 {
		for i := 1; i < len(memories); i++ {
			if memories[i-1].Importance < memories[i].Importance {
				t.Errorf("Memories not sorted by importance: %f < %f",
					memories[i-1].Importance, memories[i].Importance)
			}
		}
	}
}

func TestInMemoryVectorStore(t *testing.T) {
	store := NewInMemoryVectorStore()
	ctx := context.Background()

	// Test storing vectors
	metadata := map[string]interface{}{
		"content":  "This is a test document",
		"category": "test",
	}

	err := store.Store(ctx, "doc_1", []float64{1.0, 2.0, 3.0}, metadata)
	if err != nil {
		t.Fatalf("Failed to store vector: %v", err)
	}

	// Test searching
	results, err := store.Search(ctx, "test", 10, "test")
	if err != nil {
		t.Fatalf("Failed to search vectors: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(results))
	}

	if results[0].ID != "doc_1" {
		t.Errorf("Expected result ID 'doc_1', got %s", results[0].ID)
	}

	// Test deleting
	err = store.Delete(ctx, "doc_1")
	if err != nil {
		t.Fatalf("Failed to delete vector: %v", err)
	}

	// Verify deletion
	results, err = store.Search(ctx, "test", 10, "test")
	if err != nil {
		t.Fatalf("Failed to search after deletion: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results after deletion, got %d", len(results))
	}
}
