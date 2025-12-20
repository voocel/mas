package schema

import (
	"errors"
	"fmt"
)

var (
	// Agent-related errors
	ErrAgentNotFound        = errors.New("agent not found")
	ErrAgentNotSupported    = errors.New("agent not supported")
	ErrAgentAlreadyExists   = errors.New("agent already exists")
	ErrAgentExecutionFailed = errors.New("agent execution failed")

	// Tool-related errors
	ErrToolNotFound         = errors.New("tool not found")
	ErrToolAlreadyExists    = errors.New("tool already exists")
	ErrToolExecutionFailed  = errors.New("tool execution failed")
	ErrToolTimeout          = errors.New("tool execution timeout")
	ErrToolSandboxViolation = errors.New("tool sandbox violation")

	// LLM-related errors
	ErrModelNotSupported = errors.New("model not supported")
	ErrModelAPIError     = errors.New("model API error")
	ErrModelRateLimit    = errors.New("model rate limit exceeded")

	// Workflow-related errors (legacy)
	ErrWorkflowNotFound         = errors.New("workflow not found")
	ErrWorkflowCyclicDependency = errors.New("workflow has cyclic dependency")
	ErrWorkflowNodeNotFound     = errors.New("workflow node not found")

	// Storage-related errors
	ErrStorageNotFound  = errors.New("storage item not found")
	ErrStorageCorrupted = errors.New("storage data corrupted")

	// Common errors
	ErrInvalidInput     = errors.New("invalid input")
	ErrTimeout          = errors.New("operation timeout")
	ErrContextCancelled = errors.New("context cancelled")

	// Runner-related errors
	ErrRunnerExecutionFailed = errors.New("runner execution failed")
)

type AgentError struct {
	AgentID string
	Op      string
	Err     error
}

func (e *AgentError) Error() string {
	return fmt.Sprintf("agent %s: %s: %v", e.AgentID, e.Op, e.Err)
}

func (e *AgentError) Unwrap() error {
	return e.Err
}

func NewAgentError(agentID, op string, err error) *AgentError {
	return &AgentError{
		AgentID: agentID,
		Op:      op,
		Err:     err,
	}
}

type ToolError struct {
	ToolName string
	Op       string
	Err      error
}

func (e *ToolError) Error() string {
	return fmt.Sprintf("tool %s: %s: %v", e.ToolName, e.Op, e.Err)
}

func (e *ToolError) Unwrap() error {
	return e.Err
}

func NewToolError(toolName, op string, err error) *ToolError {
	return &ToolError{
		ToolName: toolName,
		Op:       op,
		Err:      err,
	}
}

type ModelError struct {
	Model string
	Op    string
	Err   error
}

// RunnerError describes runtime failures in Runner.
type RunnerError struct {
	Op  string
	Err error
}

func (e *RunnerError) Error() string {
	return fmt.Sprintf("runner: %s: %v", e.Op, e.Err)
}

func (e *RunnerError) Unwrap() error {
	return e.Err
}

// NewRunnerError creates a RunnerError.
func NewRunnerError(op string, err error) *RunnerError {
	return &RunnerError{Op: op, Err: err}
}

func (e *ModelError) Error() string {
	return fmt.Sprintf("model %s: %s: %v", e.Model, e.Op, e.Err)
}

func (e *ModelError) Unwrap() error {
	return e.Err
}

func NewModelError(model, op string, err error) *ModelError {
	return &ModelError{
		Model: model,
		Op:    op,
		Err:   err,
	}
}

type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field %s (value: %v): %s", e.Field, e.Value, e.Message)
}

func NewValidationError(field string, value interface{}, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

type WorkflowError struct {
	WorkflowName string
	Op           string
	Err          error
}

func (e *WorkflowError) Error() string {
	return fmt.Sprintf("workflow %s: %s: %v", e.WorkflowName, e.Op, e.Err)
}

func (e *WorkflowError) Unwrap() error {
	return e.Err
}

func NewWorkflowError(workflowName, op string, err error) *WorkflowError {
	return &WorkflowError{
		WorkflowName: workflowName,
		Op:           op,
		Err:          err,
	}
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Inspect well-known retryable errors
	switch {
	case errors.Is(err, ErrModelRateLimit):
		return true
	case errors.Is(err, ErrModelAPIError):
		return true
	case errors.Is(err, ErrTimeout):
		return true
	default:
		return false
	}
}
