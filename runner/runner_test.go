package runner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

type mockModel struct {
	calls int
}

func (m *mockModel) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	m.calls++
	if m.calls == 1 {
		msg := schema.Message{
			Role: schema.RoleAssistant,
			ToolCalls: []schema.ToolCall{
				{ID: "1", Name: "echo", Args: json.RawMessage(`{"text":"hi"}`)},
			},
		}
		return &llm.Response{Message: msg}, nil
	}
	msg := schema.Message{Role: schema.RoleAssistant, Content: "done"}
	return &llm.Response{Message: msg}, nil
}

func (m *mockModel) GenerateStream(ctx context.Context, req *llm.Request) (<-chan schema.StreamEvent, error) {
	return nil, nil
}

func (m *mockModel) SupportsTools() bool     { return true }
func (m *mockModel) SupportsStreaming() bool { return false }
func (m *mockModel) Info() llm.ModelInfo     { return llm.ModelInfo{Name: "mock"} }

type streamModel struct{}

func (m *streamModel) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return nil, nil
}

func (m *streamModel) GenerateStream(ctx context.Context, req *llm.Request) (<-chan schema.StreamEvent, error) {
	ch := make(chan schema.StreamEvent, 3)
	go func() {
		defer close(ch)
		ch <- schema.NewStreamEvent(schema.EventStart, nil)
		ch <- schema.NewTokenEvent("hello", "hello", "")
		ch <- schema.NewStreamEvent(schema.EventEnd, schema.Message{
			Role:    schema.RoleAssistant,
			Content: "done",
		})
	}()
	return ch, nil
}

func (m *streamModel) SupportsTools() bool     { return false }
func (m *streamModel) SupportsStreaming() bool { return true }
func (m *streamModel) Info() llm.ModelInfo     { return llm.ModelInfo{Name: "stream"} }

type echoTool struct {
	*tools.BaseTool
}

func newEchoTool() *echoTool {
	schema := tools.CreateToolSchema("echo", map[string]interface{}{
		"text": tools.StringProperty("text"),
	}, []string{"text"})
	return &echoTool{BaseTool: tools.NewBaseTool("echo", "echo", schema)}
}

func (t *echoTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var payload map[string]string
	_ = json.Unmarshal(input, &payload)
	return json.Marshal(map[string]string{"echo": payload["text"]})
}

type fakeExecutor struct {
	calls []schema.ToolCall
}

func (f *fakeExecutor) Execute(ctx context.Context, call schema.ToolCall, policy executor.Policy) (schema.ToolResult, error) {
	f.calls = append(f.calls, call)
	return schema.ToolResult{ID: call.ID, Result: []byte(`{"ok":true}`)}, nil
}

func (f *fakeExecutor) Close() error { return nil }

func TestRunnerToolLoop(t *testing.T) {
	model := &mockModel{}
	ag := agent.New("a1", "a1", agent.WithTools(newEchoTool()))
	r := New(Config{Model: model})

	resp, err := r.Run(context.Background(), ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "start",
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if resp.Content != "done" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
	if model.calls != 2 {
		t.Fatalf("expected 2 model calls, got %d", model.calls)
	}
}

func TestRunnerUsesToolExecutor(t *testing.T) {
	model := &mockModel{}
	exec := &fakeExecutor{}
	ag := agent.New("a1", "a1", agent.WithTools(newEchoTool()))

	r := New(Config{
		Model:        model,
		ToolExecutor: exec,
	})

	resp, err := r.Run(context.Background(), ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "start",
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if resp.Content != "done" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
	if len(exec.calls) != 1 || exec.calls[0].Name != "echo" {
		t.Fatalf("expected executor call for echo, got %v", exec.calls)
	}
}

func TestRunStreamEventIDs(t *testing.T) {
	model := &streamModel{}
	ag := agent.New("a1", "a1")
	r := New(Config{Model: model})

	ch, err := r.RunStream(context.Background(), ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "start",
	})
	if err != nil {
		t.Fatalf("run stream error: %v", err)
	}

	var events []schema.StreamEvent
	for event := range ch {
		events = append(events, event)
	}
	if len(events) == 0 {
		t.Fatal("expected stream events")
	}

	runID := events[0].RunID
	stepID := events[0].StepID
	spanID := events[0].SpanID

	if runID == "" || stepID == "" || spanID == "" {
		t.Fatalf("expected non-empty ids, got run=%q step=%q span=%q", runID, stepID, spanID)
	}

	for _, event := range events {
		if event.RunID != runID || event.StepID != stepID || event.SpanID != spanID {
			t.Fatalf("inconsistent ids in stream events")
		}
		if event.AgentID == "" {
			t.Fatalf("expected agent id on stream event")
		}
	}
}
