package policy

import (
	"context"

	"github.com/voocel/mas/executor/sandbox"
	"github.com/voocel/mas/tools"
)

type DefaultEvaluator struct {
	Registry *tools.Registry
}

func NewDefaultEvaluator(registry *tools.Registry) *DefaultEvaluator {
	return &DefaultEvaluator{Registry: registry}
}

func (e *DefaultEvaluator) Evaluate(_ context.Context, req sandbox.ExecuteToolRequest) (Decision, error) {
	if e == nil || e.Registry == nil {
		return Decision{Allowed: false, Reason: "registry is nil"}, nil
	}
	tool, ok := e.Registry.Get(req.Tool.Name)
	if !ok {
		return Decision{Allowed: false, Reason: "tool not found"}, nil
	}
	if err := ValidateToolPolicy(req.Policy, tool, req.Tool.Args); err != nil {
		return Decision{Allowed: false, Reason: err.Error()}, nil
	}
	return Decision{Allowed: true}, nil
}
