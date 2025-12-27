package local

import (
	"encoding/json"

	"github.com/voocel/mas/executor"
)

// Request represents a tool call request.
type Request struct {
	ID     string          `json:"id"`
	Tool   string          `json:"tool"`
	Args   json.RawMessage `json:"args"`
	Policy executor.Policy `json:"policy"`
}

// Response represents a tool call response.
type Response struct {
	ID       string          `json:"id"`
	Result   json.RawMessage `json:"result,omitempty"`
	Error    string          `json:"error,omitempty"`
	ExitCode int             `json:"exit_code"`
	Duration string          `json:"duration"`
}
