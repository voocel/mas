package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/local"
	"github.com/voocel/mas/tools"
	"github.com/voocel/mas/tools/builtin"
)

func main() {
	registry := tools.NewRegistry()
	registerBuiltinTools(registry)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		var req local.Request
		if err := json.Unmarshal(line, &req); err != nil {
			_ = encoder.Encode(local.Response{
				ID:       "",
				Error:    fmt.Sprintf("invalid request: %v", err),
				ExitCode: 1,
				Duration: "0s",
			})
			continue
		}

		start := time.Now()
		resp := local.Response{ID: req.ID, ExitCode: 0}

		ctx := context.Background()
		if req.Policy.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, req.Policy.Timeout)
			defer cancel()
		}

		tool, ok := registry.Get(req.Tool)
		if !ok {
			resp.ExitCode = 1
			resp.Error = "tool not found"
			resp.Duration = time.Since(start).String()
			_ = encoder.Encode(resp)
			continue
		}
		if err := validateToolPolicy(req, tool); err != nil {
			resp.ExitCode = 1
			resp.Error = err.Error()
			resp.Duration = time.Since(start).String()
			_ = encoder.Encode(resp)
			continue
		}

		result, err := tool.Execute(ctx, req.Args)
		if err != nil {
			resp.ExitCode = 1
			resp.Error = err.Error()
		} else {
			resp.Result = result
		}
		resp.Duration = time.Since(start).String()
		_ = encoder.Encode(resp)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "sandbox read error:", err)
	}
}

func registerBuiltinTools(registry *tools.Registry) {
	_ = registry.Register(builtin.NewCalculator())
	_ = registry.Register(builtin.NewFileSystemTool(nil, 0))
	_ = registry.Register(builtin.NewHTTPClientTool(0))
	_ = registry.Register(builtin.NewWebSearchTool(""))
	_ = registry.Register(builtin.NewFetchTool(0))
}

func validateToolPolicy(req local.Request, tool tools.Tool) error {
	if !isToolAllowed(req.Policy, tool) {
		return errors.New("tool not allowed")
	}
	if !req.Policy.Network.Enabled && hasCapability(tool, tools.CapabilityNetwork) {
		return errors.New("network disabled")
	}
	if tool.Name() == "file_system" {
		return validateFileAccess(req.Policy, req.Args)
	}
	if hasCapability(tool, tools.CapabilityNetwork) && len(req.Policy.Network.AllowedHosts) > 0 {
		if err := validateURLHosts(req.Policy.Network.AllowedHosts, req.Args); err != nil {
			return err
		}
	}
	return nil
}

func isToolAllowed(policy executor.Policy, tool tools.Tool) bool {
	if len(policy.AllowedTools) > 0 {
		allowed := false
		for _, name := range policy.AllowedTools {
			if name == tool.Name() {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	if len(policy.AllowedCaps) == 0 {
		return true
	}

	caps := tool.Capabilities()
	if len(caps) == 0 {
		return false
	}

	allowedCaps := make(map[tools.Capability]struct{}, len(policy.AllowedCaps))
	for _, cap := range policy.AllowedCaps {
		allowedCaps[cap] = struct{}{}
	}

	for _, cap := range caps {
		if _, ok := allowedCaps[cap]; ok {
			return true
		}
	}
	return false
}

func hasCapability(tool tools.Tool, cap tools.Capability) bool {
	for _, c := range tool.Capabilities() {
		if c == cap {
			return true
		}
	}
	return false
}

func validateFileAccess(policy executor.Policy, args json.RawMessage) error {
	path := extractPath(args)
	if path == "" {
		return errors.New("path is required")
	}

	allowed := append([]string{}, policy.AllowedPaths...)
	if policy.Workdir != "" {
		allowed = append(allowed, policy.Workdir)
	}
	if len(allowed) == 0 {
		return errors.New("path not allowed")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return errors.New("invalid path")
	}
	absPath = filepath.Clean(absPath)

	for _, p := range allowed {
		absAllowed, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		absAllowed = filepath.Clean(absAllowed)
		if absPath == absAllowed || strings.HasPrefix(absPath, absAllowed+string(os.PathSeparator)) {
			return nil
		}
	}
	return errors.New("path not allowed")
}

func validateURLHosts(allowed []string, args json.RawMessage) error {
	rawURL := extractURL(args)
	if rawURL == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return errors.New("invalid url")
	}
	host := parsed.Hostname()
	for _, allowedHost := range allowed {
		if host == allowedHost {
			return nil
		}
	}
	return errors.New("host not allowed")
}

func extractPath(args json.RawMessage) string {
	var payload map[string]any
	if err := json.Unmarshal(args, &payload); err != nil {
		return ""
	}
	if value, ok := payload["path"].(string); ok {
		return value
	}
	return ""
}

func extractURL(args json.RawMessage) string {
	var payload map[string]any
	if err := json.Unmarshal(args, &payload); err != nil {
		return ""
	}
	if value, ok := payload["url"].(string); ok {
		return value
	}
	return ""
}
