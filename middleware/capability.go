package middleware

import (
	"context"
	"fmt"

	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/tools"
)

// CapabilityRule defines capability policy.
type CapabilityRule func(caps []tools.Capability) bool

// AllowOnly allows only the specified capabilities (at least one must match).
func AllowOnly(allowed ...tools.Capability) CapabilityRule {
	allowedSet := make(map[tools.Capability]struct{}, len(allowed))
	for _, c := range allowed {
		allowedSet[c] = struct{}{}
	}
	return func(caps []tools.Capability) bool {
		for _, c := range caps {
			if _, ok := allowedSet[c]; ok {
				return true
			}
		}
		return false
	}
}

// Deny blocks the specified capabilities (any match denies).
func Deny(denied ...tools.Capability) CapabilityRule {
	deniedSet := make(map[tools.Capability]struct{}, len(denied))
	for _, c := range denied {
		deniedSet[c] = struct{}{}
	}
	return func(caps []tools.Capability) bool {
		for _, c := range caps {
			if _, ok := deniedSet[c]; ok {
				return false
			}
		}
		return true
	}
}

// ToolCapabilityPolicy blocks tool calls based on capability rules.
type ToolCapabilityPolicy struct {
	Allow CapabilityRule
	Deny  CapabilityRule
}

// NewToolCapabilityPolicy creates a capability policy middleware.
func NewToolCapabilityPolicy(allow CapabilityRule, deny CapabilityRule) *ToolCapabilityPolicy {
	return &ToolCapabilityPolicy{Allow: allow, Deny: deny}
}

func (p *ToolCapabilityPolicy) BeforeTool(ctx context.Context, state *runner.ToolState) error {
	if p == nil || state == nil || state.Agent == nil || state.Call == nil {
		return nil
	}

	var target tools.Tool
	for _, t := range state.Agent.Tools() {
		if t != nil && t.Name() == state.Call.Name {
			target = t
			break
		}
	}
	if target == nil {
		return nil
	}

	caps := target.Capabilities()
	if p.Deny != nil && !p.Deny(caps) {
		return fmt.Errorf("tool capability denied: %s", target.Name())
	}
	if p.Allow != nil && !p.Allow(caps) {
		return fmt.Errorf("tool capability not allowed: %s", target.Name())
	}
	return nil
}

var _ runner.BeforeTool = (*ToolCapabilityPolicy)(nil)
