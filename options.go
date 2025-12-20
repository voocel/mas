package mas

import (
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/tools"
)

// Option configures internal options.
type Option func(*options)

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) Option {
	return func(opts *options) {
		opts.SystemPrompt = prompt
	}
}

// WithTools sets the tool list.
func WithTools(toolList ...tools.Tool) Option {
	return func(opts *options) {
		opts.Tools = append(opts.Tools, toolList...)
	}
}

// WithMemory sets the memory store.
func WithMemory(store memory.Store) Option {
	return func(opts *options) {
		opts.Memory = store
	}
}

// WithToolInvoker sets the tool invoker.
func WithToolInvoker(invoker tools.Invoker) Option {
	return func(opts *options) {
		opts.ToolInvoker = invoker
	}
}

// WithMiddlewares sets middlewares.
func WithMiddlewares(items ...runner.Middleware) Option {
	return func(opts *options) {
		opts.Middlewares = append(opts.Middlewares, items...)
	}
}

// WithObserver sets the observer.
func WithObserver(obs runner.Observer) Option {
	return func(opts *options) {
		opts.Observer = obs
	}
}

// WithTracer sets the tracer.
func WithTracer(tracer runner.Tracer) Option {
	return func(opts *options) {
		opts.Tracer = tracer
	}
}

// WithResponseFormat sets the structured output format.
func WithResponseFormat(format *llm.ResponseFormat) Option {
	return func(opts *options) {
		opts.ResponseFormat = format
	}
}

// WithMaxTurns sets the maximum turns.
func WithMaxTurns(turns int) Option {
	return func(opts *options) {
		opts.MaxTurns = turns
	}
}

// WithHistoryWindow sets the history window.
func WithHistoryWindow(window int) Option {
	return func(opts *options) {
		opts.HistoryWindow = window
	}
}

// WithAgentID sets the agent ID.
func WithAgentID(id string) Option {
	return func(opts *options) {
		opts.AgentID = id
	}
}

// WithAgentName sets the agent name.
func WithAgentName(name string) Option {
	return func(opts *options) {
		opts.AgentName = name
	}
}

type options struct {
	SystemPrompt   string
	Tools          []tools.Tool
	Memory         memory.Store
	ToolInvoker    tools.Invoker
	Middlewares    []runner.Middleware
	Observer       runner.Observer
	Tracer         runner.Tracer
	ResponseFormat *llm.ResponseFormat
	MaxTurns       int
	HistoryWindow  int
	AgentID        string
	AgentName      string
}

func applyOptions(opts ...Option) options {
	out := options{}
	for _, opt := range opts {
		if opt != nil {
			opt(&out)
		}
	}
	return out
}
