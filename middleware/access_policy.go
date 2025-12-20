package middleware

import (
	"context"
	"fmt"

	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/tools"
)

// ToolAccessPolicy denies tool calls by default and only allows explicitly permitted tools.
type ToolAccessPolicy struct {
	AllowedNames        map[string]struct{}
	AllowedCapabilities map[tools.Capability]struct{}
}

func NewToolAccessPolicy(names []string, caps []tools.Capability) *ToolAccessPolicy {
	policy := &ToolAccessPolicy{
		AllowedNames:        make(map[string]struct{}, len(names)),
		AllowedCapabilities: make(map[tools.Capability]struct{}, len(caps)),
	}
	for _, name := range names {
		if name != "" {
			policy.AllowedNames[name] = struct{}{}
		}
	}
	for _, cap := range caps {
		policy.AllowedCapabilities[cap] = struct{}{}
	}
	return policy
}

func (p *ToolAccessPolicy) BeforeTool(_ context.Context, state *runner.ToolState) error {
	if p == nil || state == nil || state.Call == nil {
		return nil
	}
	if len(p.AllowedNames) == 0 && len(p.AllowedCapabilities) == 0 {
		return fmt.Errorf("tool not allowed: %s", state.Call.Name)
	}
	if _, ok := p.AllowedNames[state.Call.Name]; ok {
		return nil
	}
	if state.Agent == nil {
		return fmt.Errorf("tool not allowed: %s", state.Call.Name)
	}
	for _, tool := range state.Agent.Tools() {
		if tool == nil || tool.Name() != state.Call.Name {
			continue
		}
		for _, cap := range tool.Capabilities() {
			if _, ok := p.AllowedCapabilities[cap]; ok {
				return nil
			}
		}
		break
	}
	return fmt.Errorf("tool not allowed: %s", state.Call.Name)
}

var _ runner.BeforeTool = (*ToolAccessPolicy)(nil)
