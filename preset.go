package mas

// Preset returns a built-in role prompt.
func Preset(name string) string {
	if prompt, ok := presets[name]; ok {
		return prompt
	}
	return "You are a " + name + "."
}

// WithPreset applies a built-in role prompt.
func WithPreset(name string) Option {
	return WithSystemPrompt(Preset(name))
}

var presets = map[string]string{
	"assistant":  "You are a friendly assistant who provides clear, accurate, and concise answers.",
	"researcher": "You are a research assistant. Analyze the problem first, then provide a conclusion with rationale.",
	"writer":     "You are a writing assistant who produces structured, readable content.",
	"analyst":    "You are an analytical assistant who breaks down problems and delivers data-driven conclusions.",
	"engineer":   "You are an engineer who values feasibility and best practices.",
}
