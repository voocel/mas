package runner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/voocel/mas/agent"
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
