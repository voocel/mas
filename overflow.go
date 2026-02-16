package agentcore

import (
	"errors"
	"strings"
)

// contextOverflowPatterns are substrings in error messages that indicate
// the request exceeded the model's context window.
// Covers OpenAI, Anthropic, Gemini, and common proxy error formats.
var contextOverflowPatterns = []string{
	"maximum context length",
	"context length exceeded",
	"context window",
	"token limit",
	"too many tokens",
	"max_tokens",
	"maximum number of tokens",
	"input is too long",
	"prompt is too long",
	"request too large",
	"content too large",
	"exceeds the model",
	"reduce the length",
	"reduce your prompt",
}

// IsContextOverflow reports whether the error indicates a context window overflow.
// It checks for litellm validation errors (HTTP 400) with context-related keywords.
func IsContextOverflow(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	// Check for known overflow patterns in the error message
	for _, pattern := range contextOverflowPatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	// Also check the unwrapped cause chain
	var cause error = err
	for cause != nil {
		inner := errors.Unwrap(cause)
		if inner == nil {
			break
		}
		if inner == cause {
			break
		}
		cause = inner
		lowerMsg := strings.ToLower(cause.Error())
		for _, pattern := range contextOverflowPatterns {
			if strings.Contains(lowerMsg, pattern) {
				return true
			}
		}
	}

	return false
}
