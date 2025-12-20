package runner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/schema"
)

type toolLoopModel struct{}

func (m *toolLoopModel) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	msg := schema.Message{
		Role: schema.RoleAssistant,
		ToolCalls: []schema.ToolCall{
			{ID: "1", Name: "echo", Args: json.RawMessage(`{"text":"hi"}`)},
		},
	}
	return &llm.Response{Message: msg}, nil
}

func (m *toolLoopModel) GenerateStream(ctx context.Context, req *llm.Request) (<-chan schema.StreamEvent, error) {
	return nil, nil
}

func (m *toolLoopModel) SupportsTools() bool     { return true }
func (m *toolLoopModel) SupportsStreaming() bool { return false }
func (m *toolLoopModel) Info() llm.ModelInfo     { return llm.ModelInfo{Name: "loop"} }

type finalModel struct{}

func (m *finalModel) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return &llm.Response{
		Message: schema.Message{Role: schema.RoleAssistant, Content: "done"},
	}, nil
}

func (m *finalModel) GenerateStream(ctx context.Context, req *llm.Request) (<-chan schema.StreamEvent, error) {
	return nil, nil
}

func (m *finalModel) SupportsTools() bool     { return false }
func (m *finalModel) SupportsStreaming() bool { return false }
func (m *finalModel) Info() llm.ModelInfo     { return llm.ModelInfo{Name: "final"} }

func TestRunFromCheckpoint(t *testing.T) {
	ctx := context.Background()
	checkpointer := NewMemoryCheckpointer()
	ag := agent.New("a1", "a1", agent.WithTools(newEchoTool()))
	r := New(Config{
		Model:          &toolLoopModel{},
		Checkpointer:   checkpointer,
		MaxTurns:       1,
		RunIDGenerator: func() string { return "run_test" },
	})

	_, err := r.RunWithResult(ctx, ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "start",
	})
	if err == nil {
		t.Fatal("expected error due to max turns")
	}

	ckpt, err := checkpointer.Load(ctx, "run_test")
	if err != nil {
		t.Fatalf("load checkpoint error: %v", err)
	}
	if ckpt.Turn != 1 {
		t.Fatalf("expected checkpoint turn 1, got %d", ckpt.Turn)
	}

	r2 := New(Config{
		Model:    &finalModel{},
		MaxTurns: 2,
	})
	result, err := r2.RunFromCheckpoint(ctx, ag, ckpt)
	if err != nil {
		t.Fatalf("resume error: %v", err)
	}
	if result.Message.Content != "done" {
		t.Fatalf("unexpected response: %s", result.Message.Content)
	}
}
