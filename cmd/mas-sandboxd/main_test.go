package main

import (
	"testing"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/tools"
	"github.com/voocel/mas/tools/builtin"
)

func TestIsToolAllowed(t *testing.T) {
	calc := builtin.NewCalculator()
	httpTool := builtin.NewHTTPClientTool(0)

	tests := []struct {
		name   string
		policy executor.Policy
		tool   tools.Tool
		want   bool
	}{
		{"no policy", executor.Policy{}, calc, true},
		{"tool not in allowlist", executor.Policy{AllowedTools: []string{"http_client"}}, calc, false},
		{"tool in allowlist", executor.Policy{AllowedTools: []string{"calculator"}}, calc, true},
		{"caps required but none", executor.Policy{AllowedCaps: []tools.Capability{tools.CapabilityNetwork}}, calc, false},
		{"caps match", executor.Policy{AllowedCaps: []tools.Capability{tools.CapabilityNetwork}}, httpTool, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isToolAllowed(tt.policy, tt.tool); got != tt.want {
				t.Fatalf("isToolAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
