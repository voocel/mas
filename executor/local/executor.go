package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/schema"
)

// ProcessRunner starts the local sandbox process and returns its output.
type ProcessRunner interface {
	Run(ctx context.Context, path string, args []string, input []byte) ([]byte, []byte, error)
}

// ExecRunner runs the process via os/exec.
type ExecRunner struct{}

func (r ExecRunner) Run(ctx context.Context, path string, args []string, input []byte) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdin = strings.NewReader(string(input))
	out, err := cmd.CombinedOutput()
	return out, nil, err
}

// LocalExecutor executes tools via the local sandboxd process.
type LocalExecutor struct {
	Path   string
	Args   []string
	Runner ProcessRunner
}

// NewLocalExecutor creates a local executor.
func NewLocalExecutor(path string) *LocalExecutor {
	return &LocalExecutor{Path: path}
}

// Execute runs a tool call.
func (e *LocalExecutor) Execute(ctx context.Context, call schema.ToolCall, policy executor.Policy) (schema.ToolResult, error) {
	if e == nil {
		return schema.ToolResult{ID: call.ID, Error: "local executor is nil"}, errors.New("local executor is nil")
	}
	path := e.Path
	if path == "" {
		path = "mas-sandboxd"
	}
	if e.Runner == nil {
		e.Runner = ExecRunner{}
	}

	req := Request{ID: call.ID, Tool: call.Name, Args: call.Args, Policy: policy}
	data, err := json.Marshal(req)
	if err != nil {
		return schema.ToolResult{ID: call.ID, Error: err.Error()}, err
	}
	data = append(data, '\n')

	out, _, err := e.Runner.Run(ctx, path, e.Args, data)
	if err != nil {
		return schema.ToolResult{ID: call.ID, Error: err.Error()}, err
	}

	resp, err := parseResponse(out)
	if err != nil {
		return schema.ToolResult{ID: call.ID, Error: err.Error()}, err
	}

	result := schema.ToolResult{ID: call.ID, Result: resp.Result}
	if resp.Error != "" {
		result.Error = resp.Error
		return result, fmt.Errorf("sandboxd: %s", resp.Error)
	}
	return result, nil
}

// Close releases resources.
func (e *LocalExecutor) Close() error { return nil }

func parseResponse(data []byte) (*Response, error) {
	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		var resp Response
		if err := json.Unmarshal([]byte(line), &resp); err == nil {
			return &resp, nil
		}
	}
	return nil, errors.New("invalid sandboxd response")
}
