package agent

import (
	"github.com/voocel/mas/guardrail"
	"github.com/voocel/mas/tools"
)

// Option configures an Agent.
type Option func(*Config)

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(cfg *Config) {
		cfg.SystemPrompt = prompt
	}
}

// WithTools attaches tools to the agent.
func WithTools(toolList ...tools.Tool) Option {
	return func(cfg *Config) {
		cfg.Tools = append(cfg.Tools, toolList...)
	}
}

// WithMetadata sets metadata.
func WithMetadata(key string, value interface{}) Option {
	return func(cfg *Config) {
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]interface{})
		}
		cfg.Metadata[key] = value
	}
}

// WithInputGuardrails attaches input guardrails to the agent.
func WithInputGuardrails(guardrails ...guardrail.InputGuardrail) Option {
	return func(cfg *Config) {
		cfg.InputGuardrails = append(cfg.InputGuardrails, guardrails...)
	}
}

// WithOutputGuardrails attaches output guardrails to the agent.
func WithOutputGuardrails(guardrails ...guardrail.OutputGuardrail) Option {
	return func(cfg *Config) {
		cfg.OutputGuardrails = append(cfg.OutputGuardrails, guardrails...)
	}
}
