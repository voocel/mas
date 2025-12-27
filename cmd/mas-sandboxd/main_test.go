package main

import (
	"encoding/json"
	"testing"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/local"
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

func TestValidateToolPolicy(t *testing.T) {
	calc := builtin.NewCalculator()
	httpTool := builtin.NewHTTPClientTool(0)
	fileTool := builtin.NewFileSystemTool(nil, 0)

	tests := []struct {
		name    string
		req     local.Request
		tool    tools.Tool
		wantErr bool
	}{
		{
			name: "network disabled",
			req: local.Request{
				Policy: executor.Policy{Network: executor.NetworkPolicy{Enabled: false}},
				Args:   json.RawMessage(`{"url":"https://example.com"}`),
			},
			tool:    httpTool,
			wantErr: true,
		},
		{
			name: "file path denied",
			req: local.Request{
				Policy: executor.Policy{AllowedPaths: []string{"/tmp/allowed"}},
				Args:   json.RawMessage(`{"action":"read","path":"/etc/passwd"}`),
			},
			tool:    fileTool,
			wantErr: true,
		},
		{
			name: "file path allowed",
			req: local.Request{
				Policy: executor.Policy{AllowedPaths: []string{"/tmp"}},
				Args:   json.RawMessage(`{"action":"read","path":"/tmp/file.txt"}`),
			},
			tool:    fileTool,
			wantErr: false,
		},
		{
			name: "host allowed",
			req: local.Request{
				Policy: executor.Policy{Network: executor.NetworkPolicy{Enabled: true, AllowedHosts: []string{"example.com"}}},
				Args:   json.RawMessage(`{"url":"https://example.com/path"}`),
			},
			tool:    httpTool,
			wantErr: false,
		},
		{
			name: "host denied",
			req: local.Request{
				Policy: executor.Policy{Network: executor.NetworkPolicy{Enabled: true, AllowedHosts: []string{"example.com"}}},
				Args:   json.RawMessage(`{"url":"https://other.com/path"}`),
			},
			tool:    httpTool,
			wantErr: true,
		},
		{
			name: "calculator ok",
			req: local.Request{
				Policy: executor.Policy{},
				Args:   json.RawMessage(`{"expression":"1+1"}`),
			},
			tool:    calc,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateToolPolicy(tt.req, tt.tool)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateToolPolicy() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
