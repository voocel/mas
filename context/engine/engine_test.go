package engine

import (
	"context"
	"fmt"
	"testing"
	"time"

	contextpkg "github.com/voocel/mas/context"
	"github.com/voocel/mas/context/memory"
)

func TestContextEngine(t *testing.T) {
	// Create memory store and checkpointer
	memoryStore := memory.NewInMemoryStore(100)
	vectorStore := memory.NewInMemoryVectorStore()
	checkpointer := NewInMemoryCheckpointer(50)

	// Create context engine
	engine := NewContextEngine(
		WithMemory(memoryStore),
		WithVectorStore(vectorStore),
		WithCheckpointer(checkpointer),
	)

	if engine == nil {
		t.Fatal("Failed to create context engine")
	}

	// Test state creation and management
	ctx := context.Background()
	threadID := "test_thread_123"
	agentID := "test_agent"

	// Create initial state
	state := contextpkg.NewContextState(threadID, agentID)
	state.Messages = append(state.Messages, contextpkg.NewMessage("user", "Hello, world!"))

	// Test checkpoint creation
	err := engine.CreateCheckpoint(ctx, state)
	if err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Test state retrieval
	retrievedState, err := engine.GetState(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	if retrievedState.ThreadID != threadID {
		t.Errorf("Expected thread ID %s, got %s", threadID, retrievedState.ThreadID)
	}

	if len(retrievedState.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(retrievedState.Messages))
	}

	// Test state update
	update := contextpkg.StateUpdate{
		Messages: []contextpkg.Message{
			contextpkg.NewMessage("assistant", "Hello! How can I help you?"),
		},
		Scratchpad: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	err = engine.UpdateState(ctx, threadID, update)
	if err != nil {
		t.Fatalf("Failed to update state: %v", err)
	}

	// Verify update
	updatedState, err := engine.GetState(ctx, threadID)
	if err != nil {
		t.Fatalf("Failed to get updated state: %v", err)
	}

	if len(updatedState.Messages) != 2 {
		t.Errorf("Expected 2 messages after update, got %d", len(updatedState.Messages))
	}

	if updatedState.Scratchpad["test_key"] != "test_value" {
		t.Errorf("Expected scratchpad value 'test_value', got %v", updatedState.Scratchpad["test_key"])
	}
}

func TestInMemoryCheckpointer(t *testing.T) {
	checkpointer := NewInMemoryCheckpointer(10)
	ctx := context.Background()

	// Create test checkpoint
	state := contextpkg.NewContextState("thread_1", "agent_1")
	checkpoint := &contextpkg.Checkpoint{
		ID:        "checkpoint_1",
		ThreadID:  "thread_1",
		State:     state,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Test save
	err := checkpointer.Save(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Failed to save checkpoint: %v", err)
	}

	// Test load
	loaded, err := checkpointer.Load(ctx, "thread_1", "checkpoint_1")
	if err != nil {
		t.Fatalf("Failed to load checkpoint: %v", err)
	}

	if loaded.ID != "checkpoint_1" {
		t.Errorf("Expected checkpoint ID 'checkpoint_1', got %s", loaded.ID)
	}

	// Test list
	checkpoints, err := checkpointer.List(ctx, "thread_1")
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) != 1 {
		t.Errorf("Expected 1 checkpoint, got %d", len(checkpoints))
	}

	// Test delete
	err = checkpointer.Delete(ctx, "thread_1", "checkpoint_1")
	if err != nil {
		t.Fatalf("Failed to delete checkpoint: %v", err)
	}

	// Verify deletion
	checkpoints, err = checkpointer.List(ctx, "thread_1")
	if err != nil {
		t.Fatalf("Failed to list checkpoints after deletion: %v", err)
	}

	if len(checkpoints) != 0 {
		t.Errorf("Expected 0 checkpoints after deletion, got %d", len(checkpoints))
	}
}

func TestCheckpointerCleanup(t *testing.T) {
	checkpointer := NewInMemoryCheckpointer(3) // Small limit for testing
	ctx := context.Background()

	// Create multiple checkpoints
	for i := 0; i < 5; i++ {
		state := contextpkg.NewContextState("thread_1", "agent_1")
		checkpoint := &contextpkg.Checkpoint{
			ID:        fmt.Sprintf("checkpoint_%d", i),
			ThreadID:  "thread_1",
			State:     state,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Metadata:  make(map[string]interface{}),
		}

		err := checkpointer.Save(ctx, checkpoint)
		if err != nil {
			t.Fatalf("Failed to save checkpoint %d: %v", i, err)
		}
	}

	// Check that old checkpoints were cleaned up
	checkpoints, err := checkpointer.List(ctx, "thread_1")
	if err != nil {
		t.Fatalf("Failed to list checkpoints: %v", err)
	}

	if len(checkpoints) > 3 {
		t.Errorf("Expected at most 3 checkpoints after cleanup, got %d", len(checkpoints))
	}

	// Verify that the newest checkpoints are kept
	if len(checkpoints) > 0 {
		latest := checkpoints[0]
		if latest.ID != "checkpoint_4" {
			t.Errorf("Expected latest checkpoint to be 'checkpoint_4', got %s", latest.ID)
		}
	}
}

func TestStateManager(t *testing.T) {
	engine := NewContextEngine()
	stateManager := NewStateManager(engine)

	// Create test state
	state := contextpkg.NewContextState("thread_1", "agent_1")
	state.Messages = []contextpkg.Message{
		contextpkg.NewMessage("user", "This is a test message with some content"),
		contextpkg.NewMessage("assistant", "This is a response message"),
	}
	state.Scratchpad = map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}

	// Test token count calculation
	tokenCount := stateManager.CalculateTokenCount(state)
	if tokenCount <= 0 {
		t.Errorf("Expected positive token count, got %d", tokenCount)
	}

	// Test state analysis
	analysis := stateManager.AnalyzeState(state)
	if analysis.TokenCount != tokenCount {
		t.Errorf("Expected analysis token count %d, got %d", tokenCount, analysis.TokenCount)
	}

	if analysis.MessageCount != 2 {
		t.Errorf("Expected message count 2, got %d", analysis.MessageCount)
	}

	if analysis.Complexity <= 0 {
		t.Errorf("Expected positive complexity, got %f", analysis.Complexity)
	}

	// Test message filtering
	criteria := MessageFilterCriteria{
		Role:      "user",
		MinLength: 10,
	}

	filtered := stateManager.FilterMessages(state.Messages, criteria)
	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered message, got %d", len(filtered))
	}

	if filtered[0].Role != "user" {
		t.Errorf("Expected filtered message role 'user', got %s", filtered[0].Role)
	}
}
