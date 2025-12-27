package local_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/local"
	"github.com/voocel/mas/schema"
)

type fakeRunner struct {
	stdout []byte
	err    error
	input  string
}

func (r *fakeRunner) Run(_ context.Context, _ string, _ []string, input []byte) ([]byte, []byte, error) {
	r.input = string(input)
	return r.stdout, nil, r.err
}

func TestLocalExecutor_Success(t *testing.T) {
	resp := local.Response{
		ID:       "1",
		Result:   json.RawMessage(`{"ok":true}`),
		ExitCode: 0,
		Duration: "1ms",
	}
	data, _ := json.Marshal(resp)

	runner := &fakeRunner{stdout: append(data, '\n')}
	exec := &local.LocalExecutor{Path: "sandboxd", Runner: runner}

	call := schema.ToolCall{ID: "1", Name: "calc", Args: json.RawMessage(`{"expression":"1+1"}`)}
	_, err := exec.Execute(context.Background(), call, executor.Policy{AllowedTools: []string{"calc"}})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(runner.input, `"tool":"calc"`) {
		t.Fatalf("request not sent correctly: %s", runner.input)
	}
}

func TestLocalExecutor_ErrorResponse(t *testing.T) {
	resp := local.Response{
		ID:       "1",
		Error:    "tool failed",
		ExitCode: 1,
		Duration: "1ms",
	}
	data, _ := json.Marshal(resp)

	exec := &local.LocalExecutor{Runner: &fakeRunner{stdout: append(data, '\n')}}
	call := schema.ToolCall{ID: "1", Name: "calc", Args: json.RawMessage(`{}`)}
	_, err := exec.Execute(context.Background(), call, executor.Policy{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalExecutor_InvalidResponse(t *testing.T) {
	exec := &local.LocalExecutor{Runner: &fakeRunner{stdout: []byte("not json\n")}}
	call := schema.ToolCall{ID: "1", Name: "calc", Args: json.RawMessage(`{}`)}
	_, err := exec.Execute(context.Background(), call, executor.Policy{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLocalExecutor_Nil(t *testing.T) {
	var exec *local.LocalExecutor
	call := schema.ToolCall{ID: "1", Name: "calc", Args: json.RawMessage(`{}`)}
	_, err := exec.Execute(context.Background(), call, executor.Policy{})
	if err == nil || err.Error() != "local executor is nil" {
		t.Fatalf("expected nil executor error, got %v", err)
	}
}
