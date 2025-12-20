package memory

import (
	"context"
	"testing"

	"github.com/voocel/mas/schema"
)

func TestBufferAddBatchAndReset(t *testing.T) {
	store := NewBuffer(0)
	messages := []schema.Message{
		{Role: schema.RoleUser, Content: "a"},
		{Role: schema.RoleAssistant, Content: "b"},
	}

	if err := store.AddBatch(context.Background(), messages); err != nil {
		t.Fatalf("add batch error: %v", err)
	}

	history, err := store.History(context.Background())
	if err != nil {
		t.Fatalf("history error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}

	if err := store.Reset(context.Background()); err != nil {
		t.Fatalf("reset error: %v", err)
	}

	history, err = store.History(context.Background())
	if err != nil {
		t.Fatalf("history error: %v", err)
	}
	if len(history) != 0 {
		t.Fatalf("expected empty history, got %d", len(history))
	}
}
