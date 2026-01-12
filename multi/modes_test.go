package multi

import (
	"context"
	"strings"
	"testing"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/memory"
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

func TestBuildHandoffMessage(t *testing.T) {
	tests := []struct {
		name     string
		prev     schema.Message
		handoff  *schema.Handoff
		wantPre  string // expected prefix in content
		wantBody string // expected body in content
	}{
		{
			name:     "with reason and message",
			prev:     schema.Message{Content: "original"},
			handoff:  &schema.Handoff{Reason: "needs analysis", Message: "analyze this data"},
			wantPre:  "[Handoff: needs analysis]",
			wantBody: "analyze this data",
		},
		{
			name:     "message only, no reason",
			prev:     schema.Message{Content: "original"},
			handoff:  &schema.Handoff{Message: "direct message"},
			wantPre:  "",
			wantBody: "direct message",
		},
		{
			name:     "reason only, fallback to prev content",
			prev:     schema.Message{Content: "original content"},
			handoff:  &schema.Handoff{Reason: "context switch"},
			wantPre:  "[Handoff: context switch]",
			wantBody: "original content",
		},
		{
			name:     "payload message fallback",
			prev:     schema.Message{Content: "original"},
			handoff:  &schema.Handoff{Reason: "from payload", Payload: map[string]interface{}{"message": "payload msg"}},
			wantPre:  "[Handoff: from payload]",
			wantBody: "payload msg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildHandoffMessage(tt.prev, tt.handoff)
			if tt.wantPre != "" && !strings.Contains(msg.Content, tt.wantPre) {
				t.Errorf("expected prefix %q in content %q", tt.wantPre, msg.Content)
			}
			if !strings.Contains(msg.Content, tt.wantBody) {
				t.Errorf("expected body %q in content %q", tt.wantBody, msg.Content)
			}
		})
	}
}

func TestRunSequentialSharedMemory(t *testing.T) {
	r := runner.New(runner.Config{Model: &staticModel{content: "ok"}})
	a1 := agent.New("a1", "a1")
	a2 := agent.New("a2", "a2")

	shared := memory.NewBuffer(0)
	resp, err := RunSequentialWithOptions(
		context.Background(),
		r,
		[]*agent.Agent{a1, a2},
		schema.Message{Role: schema.RoleUser, Content: "hi"},
		WithSharedMemory(shared),
	)
	if err != nil {
		t.Fatalf("run error: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}

	history, err := shared.History(context.Background())
	if err != nil {
		t.Fatalf("history error: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 shared messages, got %d", len(history))
	}
}
