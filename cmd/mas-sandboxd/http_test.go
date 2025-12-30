package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/executor/sandbox/manager"
	"github.com/voocel/mas/executor/sandbox/policy"
	sandboxlocal "github.com/voocel/mas/executor/sandbox/runtime/local"
)

func TestHTTPExecute(t *testing.T) {
	rt := sandboxlocal.NewDefaultRuntime()
	evaluator := policy.NewDefaultEvaluator(rt.Registry)
	handler := newTestHandler(rt, evaluator, "token")

	req := sandbox.ExecuteToolRequest{
		ToolCallID: "call-1",
		Tool: sandbox.ToolSpec{
			Name: "calculator",
			Args: json.RawMessage(`{"expression":"1+1"}`),
		},
		Policy: executor.Policy{
			AllowedTools: []string{"calculator"},
		},
	}

	resp := doRequest(t, handler, "POST", "/v1/sandbox/execute", req, "token")
	var payload sandbox.ExecuteToolResponse
	if err := json.Unmarshal(resp, &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != sandbox.StatusOK {
		t.Fatalf("unexpected status: %s", payload.Status)
	}
	if payload.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", payload.ExitCode)
	}
}

func TestHTTPExecuteUnauthorized(t *testing.T) {
	rt := sandboxlocal.NewDefaultRuntime()
	evaluator := policy.NewDefaultEvaluator(rt.Registry)
	handler := newTestHandler(rt, evaluator, "token")

	req := sandbox.ExecuteToolRequest{
		ToolCallID: "call-1",
		Tool: sandbox.ToolSpec{
			Name: "calculator",
			Args: json.RawMessage(`{"expression":"1+1"}`),
		},
		Policy: executor.Policy{
			AllowedTools: []string{"calculator"},
		},
	}

	res := doRawRequest(t, handler, "POST", "/v1/sandbox/execute", req, "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", res.StatusCode)
	}
}

func newTestHandler(rt *sandboxlocal.Runtime, evaluator policy.Evaluator, token string) http.Handler {
	svc := &manager.Service{Runtime: rt, Evaluator: evaluator}
	return newHTTPHandler(token, svc)
}

func doRequest(t *testing.T, handler http.Handler, method, path string, payload any, token string) []byte {
	t.Helper()
	res := doRawRequest(t, handler, method, path, payload, token)
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d: %s", res.StatusCode, string(data))
	}
	return data
}

func doRawRequest(t *testing.T, handler http.Handler, method, path string, payload any, token string) *http.Response {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req = req.WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Result()
}
