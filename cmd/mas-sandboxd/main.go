package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
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
		if !isToolAllowed(req.Policy, tool) {
			resp.ExitCode = 1
			resp.Error = "tool not allowed"
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
		fmt.Fprintln(os.Stderr, "sandboxd read error:", err)
	}
}

func registerBuiltinTools(registry *tools.Registry) {
	_ = registry.Register(builtin.NewCalculator())
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
