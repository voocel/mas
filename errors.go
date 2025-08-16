package mas

import (
	"errors"
	"fmt"
)

// Core error types
var (
	// Agent errors
	ErrAgentNotConfigured   = errors.New("agent not properly configured")
	ErrNoLLMProvider       = errors.New("no LLM provider configured")
	ErrToolNotFound        = errors.New("tool not found")
	ErrToolExecutionFailed = errors.New("tool execution failed")
	
	// Workflow errors
	ErrWorkflowNotStarted     = errors.New("workflow not started")
	ErrNodeNotFound          = errors.New("workflow node not found")
	ErrWorkflowCycle         = errors.New("workflow contains cycles")
	ErrInvalidWorkflowState  = errors.New("invalid workflow state")
	
	// Memory errors
	ErrMemoryFull           = errors.New("memory capacity exceeded")
	ErrInvalidMemoryConfig  = errors.New("invalid memory configuration")
	
	// Checkpoint errors
	ErrCheckpointNotFound   = errors.New("checkpoint not found")
	ErrCheckpointCorrupted  = errors.New("checkpoint data corrupted")
	ErrCheckpointVersion    = errors.New("incompatible checkpoint version")
	
	// Provider errors
	ErrProviderUnavailable  = errors.New("LLM provider unavailable")
	ErrInvalidModel         = errors.New("invalid model name")
	ErrAPIKeyRequired       = errors.New("API key required")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
	
	// Tool errors
	ErrInvalidToolSchema    = errors.New("invalid tool schema")
	ErrToolParameterInvalid = errors.New("invalid tool parameters")
	ErrToolPermissionDenied = errors.New("tool permission denied")
)

// Error wrapping helpers

// WrapAgentError wraps an agent-related error with context
func WrapAgentError(err error, msg string) error {
	return fmt.Errorf("agent error: %s: %w", msg, err)
}

// WrapWorkflowError wraps a workflow-related error with context
func WrapWorkflowError(err error, msg string) error {
	return fmt.Errorf("workflow error: %s: %w", msg, err)
}

// WrapMemoryError wraps a memory-related error with context
func WrapMemoryError(err error, msg string) error {
	return fmt.Errorf("memory error: %s: %w", msg, err)
}

// WrapCheckpointError wraps a checkpoint-related error with context
func WrapCheckpointError(err error, msg string) error {
	return fmt.Errorf("checkpoint error: %s: %w", msg, err)
}

// WrapProviderError wraps a provider-related error with context
func WrapProviderError(err error, msg string) error {
	return fmt.Errorf("provider error: %s: %w", msg, err)
}

// WrapToolError wraps a tool-related error with context
func WrapToolError(err error, msg string) error {
	return fmt.Errorf("tool error: %s: %w", msg, err)
}