package multi

import (
	"context"
	"testing"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

type staticModel struct {
	content string
}

func (m *staticModel) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return &llm.Response{Message: schema.Message{Role: schema.RoleAssistant, Content: m.content}}, nil
}

func (m *staticModel) GenerateStream(ctx context.Context, req *llm.Request) (<-chan schema.StreamEvent, error) {
	return nil, nil
}

func (m *staticModel) SupportsTools() bool     { return false }
func (m *staticModel) SupportsStreaming() bool { return false }
func (m *staticModel) Info() llm.ModelInfo     { return llm.ModelInfo{Name: "static"} }

func TestRunSequential(t *testing.T) {
	r := runner.New(runner.Config{Model: &staticModel{content: "ok"}})
	a1 := agent.New("a1", "a1")
	a2 := agent.New("a2", "a2")

	resp, err := RunSequential(context.Background(), r, []*agent.Agent{a1, a2}, schema.Message{
		Role:    schema.RoleUser,
		Content: "hi",
	})
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
}

func TestRunParallel(t *testing.T) {
	r := runner.New(runner.Config{Model: &staticModel{content: "ok"}})
	a1 := agent.New("a1", "a1")
	a2 := agent.New("a2", "a2")

	resp, err := RunParallel(context.Background(), r, []*agent.Agent{a1, a2}, schema.Message{
		Role:    schema.RoleUser,
		Content: "hi",
	}, nil)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
}
