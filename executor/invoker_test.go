package executor_test

import (
	"context"
	"testing"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

type fakeExecutor struct {
	calls  []schema.ToolCall
	policy executor.Policy
	result schema.ToolResult
}

func (f *fakeExecutor) Execute(_ context.Context, call schema.ToolCall, policy executor.Policy) (schema.ToolResult, error) {
	f.calls = append(f.calls, call)
	f.policy = policy
	return f.result, nil
}

func (f *fakeExecutor) Close() error { return nil }

func TestExecutorInvoker_UsesExecutor(t *testing.T) {
	exec := &fakeExecutor{
		result: schema.ToolResult{ID: "1", Result: []byte(`{"ok":true}`)},
	}

	invoker := &executor.ExecutorInvoker{
		Executor: exec,
		Policy: executor.Policy{
			AllowedTools: []string{"calc"},
		},
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tools.NewBaseTool("calc", "test tool", nil)); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	calls := []schema.ToolCall{
		{ID: "1", Name: "calc", Args: []byte(`{}`)},
	}

	results, err := invoker.Invoke(context.Background(), registry, calls)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Invoke() results len = %d, want 1", len(results))
	}
	if len(exec.calls) != 1 || exec.calls[0].Name != "calc" {
		t.Fatalf("executor not called correctly: %v", exec.calls)
	}
	if got := exec.policy.AllowedTools; len(got) != 1 || got[0] != "calc" {
		t.Fatalf("policy not passed through: %v", exec.policy)
	}
}
