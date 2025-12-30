package main

import (
	"encoding/json"
	"testing"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/sandbox/policy"
	"github.com/voocel/mas/tools"
	"github.com/voocel/mas/tools/builtin"
)

func TestValidateToolPolicy(t *testing.T) {
	calc := builtin.NewCalculator()
	httpTool := builtin.NewHTTPClientTool(0)
	fileTool := builtin.NewFileSystemTool(nil, 0)

	tests := []struct {
		name    string
		policy  executor.Policy
		tool    tools.Tool
		args    json.RawMessage
		wantErr bool
	}{
		{
			name:    "tool not in allowlist",
			policy:  executor.Policy{AllowedTools: []string{"http_client"}},
			tool:    calc,
			args:    json.RawMessage(`{"expression":"1+1"}`),
			wantErr: true,
		},
		{
			name:    "tool in allowlist",
			policy:  executor.Policy{AllowedTools: []string{"calculator"}},
			tool:    calc,
			args:    json.RawMessage(`{"expression":"1+1"}`),
			wantErr: false,
		},
		{
			name:    "caps required but none",
			policy:  executor.Policy{AllowedCaps: []tools.Capability{tools.CapabilityNetwork}},
			tool:    calc,
			args:    json.RawMessage(`{"expression":"1+1"}`),
			wantErr: true,
		},
		{
			name: "caps match",
			policy: executor.Policy{
				AllowedCaps: []tools.Capability{tools.CapabilityNetwork},
				Network:     executor.NetworkPolicy{Enabled: true},
			},
			tool:    httpTool,
			args:    json.RawMessage(`{"url":"https://example.com"}`),
			wantErr: false,
		},
		{
			name:    "caps mismatch",
			policy:  executor.Policy{AllowedCaps: []tools.Capability{tools.CapabilityFile}, Network: executor.NetworkPolicy{Enabled: true}},
			tool:    httpTool,
			args:    json.RawMessage(`{"url":"https://example.com"}`),
			wantErr: true,
		},
		{
			name:    "network disabled",
			policy:  executor.Policy{Network: executor.NetworkPolicy{Enabled: false}},
			tool:    httpTool,
			args:    json.RawMessage(`{"url":"https://example.com"}`),
			wantErr: true,
		},
		{
			name:    "file path denied",
			policy:  executor.Policy{AllowedPaths: []string{"/tmp/allowed"}},
			tool:    fileTool,
			args:    json.RawMessage(`{"action":"read","path":"/etc/passwd"}`),
			wantErr: true,
		},
		{
			name:    "file path allowed",
			policy:  executor.Policy{AllowedPaths: []string{"/tmp"}},
			tool:    fileTool,
			args:    json.RawMessage(`{"action":"read","path":"/tmp/file.txt"}`),
			wantErr: false,
		},
		{
			name:    "host allowed",
			policy:  executor.Policy{Network: executor.NetworkPolicy{Enabled: true, AllowedHosts: []string{"example.com"}}},
			tool:    httpTool,
			args:    json.RawMessage(`{"url":"https://example.com/path"}`),
			wantErr: false,
		},
		{
			name:    "host denied",
			policy:  executor.Policy{Network: executor.NetworkPolicy{Enabled: true, AllowedHosts: []string{"example.com"}}},
			tool:    httpTool,
			args:    json.RawMessage(`{"url":"https://other.com/path"}`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.ValidateToolPolicy(tt.policy, tt.tool, tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateToolPolicy() err=%v wantErr=%v", err, tt.wantErr)
			}
		})
	}
}
