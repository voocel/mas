package mas

import "context"

// AgentOption configures an Agent.
type AgentOption func(*Agent)

// WithModel sets the LLM model.
func WithModel(model ChatModel) AgentOption {
	return func(a *Agent) { a.model = model }
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) AgentOption {
	return func(a *Agent) { a.systemPrompt = prompt }
}

// WithTools sets the tool list.
func WithTools(tools ...Tool) AgentOption {
	return func(a *Agent) { a.tools = tools }
}

// WithMaxTurns sets the max turns safety limit.
func WithMaxTurns(n int) AgentOption {
	return func(a *Agent) { a.maxTurns = n }
}

// WithStreamFn sets a custom LLM call function (for proxy/mock).
func WithStreamFn(fn StreamFn) AgentOption {
	return func(a *Agent) { a.streamFn = fn }
}

// WithTransformContext sets the context transform function.
func WithTransformContext(fn func(ctx context.Context, msgs []AgentMessage) ([]AgentMessage, error)) AgentOption {
	return func(a *Agent) { a.transformContext = fn }
}

// WithConvertToLLM sets the message conversion function.
func WithConvertToLLM(fn func([]AgentMessage) []Message) AgentOption {
	return func(a *Agent) { a.convertToLLM = fn }
}

// WithSteeringMode sets the steering queue drain mode.
// QueueModeAll (default) delivers all queued steering messages at once.
// QueueModeOneAtATime delivers one per turn, letting the agent respond to each individually.
func WithSteeringMode(mode QueueMode) AgentOption {
	return func(a *Agent) { a.steeringMode = mode }
}

// WithFollowUpMode sets the follow-up queue drain mode.
// QueueModeAll (default) delivers all queued follow-up messages at once.
// QueueModeOneAtATime delivers one per turn.
func WithFollowUpMode(mode QueueMode) AgentOption {
	return func(a *Agent) { a.followUpMode = mode }
}
