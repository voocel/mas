package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

type errModel struct{}

func (m *errModel) Generate(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	return nil, schema.ErrModelRateLimit
}

func (m *errModel) GenerateStream(ctx context.Context, req *llm.Request) (<-chan schema.StreamEvent, error) {
	return nil, nil
}

func (m *errModel) SupportsTools() bool     { return false }
func (m *errModel) SupportsStreaming() bool { return false }
func (m *errModel) Info() llm.ModelInfo     { return llm.ModelInfo{Name: "err"} }

func TestRetryMiddleware(t *testing.T) {
	mw := &RetryMiddleware{MaxAttempts: 2, BaseDelay: 1 * time.Millisecond}
	_, err := mw.HandleLLM(context.Background(), &runner.State{}, &llm.Request{}, func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return nil, schema.ErrModelRateLimit
	})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	mw := &TimeoutMiddleware{LLMTimeout: 10 * time.Millisecond}
	ctx, cancel := mw.LLMContext(context.Background(), &runner.State{})
	if cancel == nil {
		t.Fatalf("expected cancel")
	}
	cancel()
	select {
	case <-ctx.Done():
	default:
		t.Fatalf("expected context done")
	}
}

type capTool struct {
	*tools.BaseTool
}

func newCapTool() *capTool {
	return &capTool{BaseTool: tools.NewBaseTool("file_tool", "file", nil).WithCapabilities(tools.CapabilityFile)}
}

func TestCapabilityPolicy(t *testing.T) {
	ag := agent.New("a1", "a1", agent.WithTools(newCapTool()))
	policy := NewToolCapabilityPolicy(nil, Deny(tools.CapabilityFile))

	state := &runner.ToolState{
		Agent: ag,
		Call:  &schema.ToolCall{Name: "file_tool"},
	}

	if err := policy.BeforeTool(context.Background(), state); err == nil {
		t.Fatalf("expected capability denied error")
	}
}

func TestToolAccessPolicyDenyByDefault(t *testing.T) {
	ag := agent.New("a1", "a1", agent.WithTools(newCapTool()))
	policy := NewToolAccessPolicy(nil, nil)

	state := &runner.ToolState{
		Agent: ag,
		Call:  &schema.ToolCall{Name: "file_tool"},
	}

	if err := policy.BeforeTool(context.Background(), state); err == nil {
		t.Fatalf("expected default deny error")
	}
}

func TestToolAccessPolicyAllowByName(t *testing.T) {
	ag := agent.New("a1", "a1", agent.WithTools(newCapTool()))
	policy := NewToolAccessPolicy([]string{"file_tool"}, nil)

	state := &runner.ToolState{
		Agent: ag,
		Call:  &schema.ToolCall{Name: "file_tool"},
	}

	if err := policy.BeforeTool(context.Background(), state); err != nil {
		t.Fatalf("expected allow by name, got %v", err)
	}
}

func TestToolAccessPolicyAllowByCapability(t *testing.T) {
	ag := agent.New("a1", "a1", agent.WithTools(newCapTool()))
	policy := NewToolAccessPolicy(nil, []tools.Capability{tools.CapabilityFile})

	state := &runner.ToolState{
		Agent: ag,
		Call:  &schema.ToolCall{Name: "file_tool"},
	}

	if err := policy.BeforeTool(context.Background(), state); err != nil {
		t.Fatalf("expected allow by capability, got %v", err)
	}
}

func TestHITLInterruptLLM(t *testing.T) {
	mw := &HITLMiddleware{
		Approver: HITLFunc{
			LLM: func(ctx context.Context, state *runner.State, req *llm.Request) HITLDecision {
				return Interrupt("need review")
			},
		},
	}

	_, err := mw.HandleLLM(context.Background(), &runner.State{}, &llm.Request{}, func(ctx context.Context, req *llm.Request) (*llm.Response, error) {
		return &llm.Response{Message: schema.Message{Role: schema.RoleAssistant}}, nil
	})
	if err == nil || !IsHITLInterrupt(err) {
		t.Fatalf("expected hitl interrupt error")
	}
}

func TestHITLInterruptTool(t *testing.T) {
	mw := &HITLMiddleware{
		Approver: HITLFunc{
			Tool: func(ctx context.Context, state *runner.ToolState) HITLDecision {
				return Interrupt("tool needs approval")
			},
		},
	}

	err := mw.BeforeTool(context.Background(), &runner.ToolState{Call: &schema.ToolCall{Name: "echo"}})
	if err == nil || !IsHITLInterrupt(err) {
		t.Fatalf("expected hitl interrupt error")
	}
}
