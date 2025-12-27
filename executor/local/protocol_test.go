package local_test

import (
	"encoding/json"
	"testing"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/local"
)

func TestProtocol_RoundTrip(t *testing.T) {
	req := local.Request{
		ID:   "1",
		Tool: "calc",
		Args: json.RawMessage(`{"expression":"1+1"}`),
		Policy: executor.Policy{
			AllowedTools: []string{"calc"},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got local.Request
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != req.ID || got.Tool != req.Tool {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}
