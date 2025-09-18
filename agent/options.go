package agent

import (
	"github.com/voocel/mas/tools"
)

// Option defines an agent configuration option.
type Option func(*AgentConfig)

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(config *AgentConfig) {
		config.SystemPrompt = prompt
	}
}

// WithTools sets the list of tools.
func WithTools(toolList ...tools.Tool) Option {
	return func(config *AgentConfig) {
		config.Tools = append(config.Tools, toolList...)
	}
}

// WithMaxHistory sets the maximum number of history records.
func WithMaxHistory(maxHistory int) Option {
	return func(config *AgentConfig) {
		if maxHistory > 0 {
			config.MaxHistory = maxHistory
		}
	}
}

// WithTemperature sets the temperature parameter.
func WithTemperature(temperature float64) Option {
	return func(config *AgentConfig) {
		if temperature >= 0.0 && temperature <= 2.0 {
			config.Temperature = temperature
		}
	}
}

// WithMaxTokens sets the maximum number of tokens.
func WithMaxTokens(maxTokens int) Option {
	return func(config *AgentConfig) {
		if maxTokens > 0 {
			config.MaxTokens = maxTokens
		}
	}
}

// WithCalculator adds the calculator tool.
func WithCalculator() Option {
	return func(config *AgentConfig) {
		// Directly import the builtin package to create a calculator.
		// calculator := builtin.NewCalculator()
		// config.Tools = append(config.Tools, calculator)
		// Skip for now to avoid circular imports.
	}
}

// WithToolRegistry uses an existing tool registry.
func WithToolRegistry(registry *tools.Registry) Option {
	return func(config *AgentConfig) {
		if registry != nil {
			config.Tools = append(config.Tools, registry.List()...)
		}
	}
}

// WithBuiltinTools adds all built-in tools.
func WithBuiltinTools() Option {
	return func(config *AgentConfig) {
		// Skip built-in tools for now to avoid circular imports.
		// TODO: Refactor the tool creation method.
	}
}

// WithRole sets the agent's role (via system prompt).
func WithRole(role string) Option {
	rolePrompts := map[string]string{
		"assistant":  "You are a helpful AI assistant. You provide accurate, helpful, and friendly responses to user questions.",
		"researcher": "You are a research specialist. You excel at finding, analyzing, and synthesizing information from various sources.",
		"writer":     "You are a professional writer. You create clear, engaging, and well-structured content tailored to the audience.",
		"analyst":    "You are a data analyst. You excel at interpreting data, identifying patterns, and providing actionable insights.",
		"teacher":    "You are an educational assistant. You explain concepts clearly, provide examples, and help users learn effectively.",
		"developer":  "You are a software development assistant. You help with coding, debugging, architecture design, and best practices.",
		"consultant": "You are a business consultant. You provide strategic advice, problem-solving guidance, and professional recommendations.",
	}

	return func(config *AgentConfig) {
		if prompt, exists := rolePrompts[role]; exists {
			config.SystemPrompt = prompt
		} else {
			// If the role is not in the predefined list, use it as a custom role.
			config.SystemPrompt = "You are a " + role + ". Please act according to this role and provide appropriate responses."
		}
	}
}

// WithPersonality sets the agent's personality traits.
func WithPersonality(traits ...string) Option {
	return func(config *AgentConfig) {
		if len(traits) == 0 {
			return
		}

		personalityPrompt := "You have the following personality traits: "
		for i, trait := range traits {
			if i > 0 {
				personalityPrompt += ", "
			}
			personalityPrompt += trait
		}
		personalityPrompt += ". Please reflect these traits in your responses while remaining helpful and professional."

		if config.SystemPrompt != "" {
			config.SystemPrompt += "\n\n" + personalityPrompt
		} else {
			config.SystemPrompt = personalityPrompt
		}
	}
}

// WithExpertise sets the areas of expertise.
func WithExpertise(domains ...string) Option {
	return func(config *AgentConfig) {
		if len(domains) == 0 {
			return
		}

		expertisePrompt := "You are an expert in the following domains: "
		for i, domain := range domains {
			if i > 0 {
				expertisePrompt += ", "
			}
			expertisePrompt += domain
		}
		expertisePrompt += ". Use your expertise to provide detailed, accurate, and insightful responses in these areas."

		if config.SystemPrompt != "" {
			config.SystemPrompt += "\n\n" + expertisePrompt
		} else {
			config.SystemPrompt = expertisePrompt
		}
	}
}

// WithConstraints sets behavioral constraints.
func WithConstraints(constraints ...string) Option {
	return func(config *AgentConfig) {
		if len(constraints) == 0 {
			return
		}

		constraintPrompt := "Please follow these constraints: "
		for i, constraint := range constraints {
			if i > 0 {
				constraintPrompt += "; "
			}
			constraintPrompt += constraint
		}
		constraintPrompt += "."

		if config.SystemPrompt != "" {
			config.SystemPrompt += "\n\n" + constraintPrompt
		} else {
			config.SystemPrompt = constraintPrompt
		}
	}
}

// WithOutputFormat sets the output format requirement.
func WithOutputFormat(format string) Option {
	formatPrompts := map[string]string{
		"json":     "Always respond in valid JSON format.",
		"markdown": "Format your responses using Markdown syntax.",
		"xml":      "Structure your responses in XML format.",
		"yaml":     "Format your responses in YAML format.",
		"plain":    "Provide responses in plain text without special formatting.",
		"bullet":   "Use bullet points to structure your responses.",
		"numbered": "Use numbered lists to organize your responses.",
	}

	return func(config *AgentConfig) {
		var formatPrompt string
		if prompt, exists := formatPrompts[format]; exists {
			formatPrompt = prompt
		} else {
			formatPrompt = "Format your responses according to: " + format
		}

		if config.SystemPrompt != "" {
			config.SystemPrompt += "\n\n" + formatPrompt
		} else {
			config.SystemPrompt = formatPrompt
		}
	}
}

// WithLanguage sets the response language.
func WithLanguage(language string) Option {
	return func(config *AgentConfig) {
		languagePrompt := "Please respond in " + language + " language."

		if config.SystemPrompt != "" {
			config.SystemPrompt += "\n\n" + languagePrompt
		} else {
			config.SystemPrompt = languagePrompt
		}
	}
}

// WithContext adds context information.
func WithContext(contextInfo string) Option {
	return func(config *AgentConfig) {
		contextPrompt := "Context: " + contextInfo

		if config.SystemPrompt != "" {
			config.SystemPrompt = contextPrompt + "\n\n" + config.SystemPrompt
		} else {
			config.SystemPrompt = contextPrompt
		}
	}
}

// CombineOptions combines multiple options.
func CombineOptions(options ...Option) Option {
	return func(config *AgentConfig) {
		for _, option := range options {
			option(config)
		}
	}
}

// PresetAssistant is a preset assistant configuration.
func PresetAssistant() Option {
	return CombineOptions(
		WithRole("assistant"),
		WithPersonality("helpful", "friendly", "patient"),
		WithMaxHistory(20),
		WithTemperature(0.7),
	)
}

// PresetResearcher is a preset researcher configuration.
func PresetResearcher() Option {
	return CombineOptions(
		WithRole("researcher"),
		WithPersonality("analytical", "thorough", "objective"),
		WithMaxHistory(30),
		WithTemperature(0.3),
		WithBuiltinTools(),
	)
}

// PresetWriter is a preset writer assistant configuration.
func PresetWriter() Option {
	return CombineOptions(
		WithRole("writer"),
		WithPersonality("creative", "articulate", "detail-oriented"),
		WithMaxHistory(25),
		WithTemperature(0.8),
		WithOutputFormat("markdown"),
	)
}

// PresetAnalyst is a preset analyst configuration.
func PresetAnalyst() Option {
	return CombineOptions(
		WithRole("analyst"),
		WithPersonality("logical", "precise", "data-driven"),
		WithMaxHistory(15),
		WithTemperature(0.2),
		WithCalculator(),
	)
}

// WithCapabilities sets the agent's capability declaration.
func WithCapabilities(capabilities *AgentCapabilities) Option {
	return func(config *AgentConfig) {
		config.Capabilities = capabilities
	}
}

// WithCoreCapabilities sets the core capabilities.
func WithCoreCapabilities(capabilities ...Capability) Option {
	return func(config *AgentConfig) {
		if config.Capabilities == nil {
			config.Capabilities = &AgentCapabilities{}
		}
		config.Capabilities.CoreCapabilities = capabilities
	}
}

// WithExpertiseDomains sets the areas of expertise.
func WithExpertiseDomains(domains ...string) Option {
	return func(config *AgentConfig) {
		if config.Capabilities == nil {
			config.Capabilities = &AgentCapabilities{}
		}
		config.Capabilities.Expertise = domains
	}
}

// WithComplexityLevel sets the processing complexity.
func WithComplexityLevel(level int) Option {
	return func(config *AgentConfig) {
		if config.Capabilities == nil {
			config.Capabilities = &AgentCapabilities{}
		}
		if level >= 1 && level <= 10 {
			config.Capabilities.ComplexityLevel = level
		}
	}
}

// WithConcurrencyLevel sets the concurrent processing capability.
func WithConcurrencyLevel(level int) Option {
	return func(config *AgentConfig) {
		if config.Capabilities == nil {
			config.Capabilities = &AgentCapabilities{}
		}
		if level >= 1 {
			config.Capabilities.ConcurrencyLevel = level
		}
	}
}

// WithSupportedLanguages sets the supported languages.
func WithSupportedLanguages(languages ...string) Option {
	return func(config *AgentConfig) {
		if config.Capabilities == nil {
			config.Capabilities = &AgentCapabilities{}
		}
		config.Capabilities.Languages = languages
	}
}

// WithCustomTags sets custom capability tags.
func WithCustomTags(tags ...string) Option {
	return func(config *AgentConfig) {
		if config.Capabilities == nil {
			config.Capabilities = &AgentCapabilities{}
		}
		config.Capabilities.CustomTags = tags
	}
}

// WithCapabilityDescription sets the capability description.
func WithCapabilityDescription(description string) Option {
	return func(config *AgentConfig) {
		if config.Capabilities == nil {
			config.Capabilities = &AgentCapabilities{}
		}
		config.Capabilities.Description = description
	}
}
