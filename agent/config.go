package agent

import (
	"fmt"
)

type ConfigOption func(*AgentConfig)

func WithName(name string) ConfigOption {
	return func(c *AgentConfig) {
		c.Name = name
	}
}

func WithModel(model string) ConfigOption {
	return func(c *AgentConfig) {
		c.Model = model
	}
}

func WithAPIKey(apiKey string) ConfigOption {
	return func(c *AgentConfig) {
		c.APIKey = apiKey
	}
}

func WithSystemPrompt(prompt string) ConfigOption {
	return func(c *AgentConfig) {
		c.SystemPrompt = prompt
	}
}

func WithTemperature(temp float64) ConfigOption {
	return func(c *AgentConfig) {
		c.Temperature = temp
	}
}

func WithMaxTokens(tokens int) ConfigOption {
	return func(c *AgentConfig) {
		c.MaxTokens = tokens
	}
}

func WithTools(tools ...Tool) ConfigOption {
	return func(c *AgentConfig) {
		c.Tools = tools
	}
}

func WithMemory(memory Memory) ConfigOption {
	return func(c *AgentConfig) {
		c.Memory = memory
	}
}

func ApplyOptions(config *AgentConfig, options ...ConfigOption) {
	for _, option := range options {
		option(config)
	}
}

func ValidateConfig(config AgentConfig) error {
	if config.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}

	if config.APIKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	if config.Temperature < 0 || config.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if config.MaxTokens < 1 {
		return fmt.Errorf("max tokens must be greater than 0")
	}

	return nil
}

func NewChatAgentConfig(model, apiKey string) AgentConfig {
	config := DefaultAgentConfig()
	config.Model = model
	config.APIKey = apiKey
	config.Temperature = 0.7
	config.MaxTokens = 2000
	return config
}

func NewAnalysisAgentConfig(model, apiKey string) AgentConfig {
	config := DefaultAgentConfig()
	config.Model = model
	config.APIKey = apiKey
	config.Temperature = 0.1 // Lower temperature for more consistent analysis
	config.MaxTokens = 4000  // Higher token limit for detailed analysis
	return config
}

func NewCreativeAgentConfig(model, apiKey string) AgentConfig {
	config := DefaultAgentConfig()
	config.Model = model
	config.APIKey = apiKey
	config.Temperature = 1.0 // Higher temperature for more creativity
	config.MaxTokens = 3000
	return config
}
