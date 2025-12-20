package agent

import "github.com/voocel/mas/tools"

// Config defines a lightweight agent configuration.
type Config struct {
	ID           string
	Name         string
	SystemPrompt string
	Tools        []tools.Tool
	Metadata     map[string]interface{}
}

// Agent is a lightweight descriptor and does not execute tools or call models.
type Agent struct {
	config Config
}

// New creates an Agent with options.
func New(id, name string, opts ...Option) *Agent {
	cfg := Config{
		ID:   id,
		Name: name,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return &Agent{config: cfg}
}

// NewWithConfig creates an Agent from a config struct.
func NewWithConfig(cfg Config) *Agent {
	return &Agent{config: cfg}
}

func (a *Agent) ID() string {
	return a.config.ID
}

func (a *Agent) Name() string {
	return a.config.Name
}

func (a *Agent) SystemPrompt() string {
	return a.config.SystemPrompt
}

func (a *Agent) Tools() []tools.Tool {
	return append([]tools.Tool(nil), a.config.Tools...)
}

func (a *Agent) Metadata() map[string]interface{} {
	if a.config.Metadata == nil {
		return nil
	}
	cp := make(map[string]interface{}, len(a.config.Metadata))
	for k, v := range a.config.Metadata {
		cp[k] = v
	}
	return cp
}
