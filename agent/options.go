package agent

import "github.com/voocel/mas/tools"

// Option configures an agent when constructing it.
type Option func(*AgentConfig)

// WithSystemPrompt replaces the default system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(cfg *AgentConfig) {
		cfg.SystemPrompt = prompt
	}
}

// WithTools registers additional tools for the agent.
func WithTools(toolList ...tools.Tool) Option {
	return func(cfg *AgentConfig) {
		cfg.Tools = append(cfg.Tools, toolList...)
	}
}

// WithHistoryWindow controls how many prior messages are sent to the model.
func WithHistoryWindow(window int) Option {
	return func(cfg *AgentConfig) {
		if window > 0 {
			cfg.HistoryWindow = window
		}
	}
}

// WithTemperature overrides the default sampling temperature.
func WithTemperature(temp float64) Option {
	return func(cfg *AgentConfig) {
		if temp >= 0.0 && temp <= 2.0 {
			cfg.Temperature = temp
		}
	}
}

// WithMaxTokens sets an upper bound for response tokens.
func WithMaxTokens(maxTokens int) Option {
	return func(cfg *AgentConfig) {
		if maxTokens > 0 {
			cfg.MaxTokens = maxTokens
		}
	}
}

// WithPresetRole provides common role prompts without forcing users to craft them manually.
func WithPresetRole(role string) Option {
	rolePrompts := map[string]string{
		"assistant":  "You are a helpful AI assistant. Provide accurate, helpful, and friendly responses.",
		"researcher": "You are an analytical researcher. Gather, compare, and synthesise reliable information.",
		"writer":     "You are a professional writer. Produce engaging, well-structured content tailored to the audience.",
		"analyst":    "You are a data analyst. Interpret data, highlight patterns, and suggest data-driven actions.",
		"developer":  "You are a pragmatic software engineer. Offer clear explanations, code snippets, and best practices.",
	}

	return func(cfg *AgentConfig) {
		if prompt, ok := rolePrompts[role]; ok {
			cfg.SystemPrompt = prompt
		} else {
			cfg.SystemPrompt = "You are a " + role + ". Respond accordingly with clarity and professionalism."
		}
	}
}
