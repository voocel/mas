package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/voocel/mas/executor/local"
	"github.com/voocel/mas/executor/sandbox/policy"
	sandboxruntime "github.com/voocel/mas/executor/sandbox/runtime"
	sandboxlocal "github.com/voocel/mas/executor/sandbox/runtime/local"
	sandboxmicrovm "github.com/voocel/mas/executor/sandbox/runtime/microvm"
	"github.com/voocel/mas/tools"
	"github.com/voocel/mas/tools/builtin"
)

func main() {
	listen := flag.String("listen", "", "http listen address for sandbox manager")
	authToken := flag.String("auth-token", "", "optional shared token for http api")
	runtimeType := flag.String("runtime", "local", "runtime backend: local or microvm")
	runtimeConfig := flag.String("runtime-config", "", "runtime config file path")
	flag.Parse()

	if *listen != "" {
		registry := tools.NewRegistry()
		registerBuiltinTools(registry)

		evaluator := policy.NewDefaultEvaluator(registry)
		runtime, err := buildRuntime(*runtimeType, *runtimeConfig, registry)
		if err != nil {
			fmt.Fprintln(os.Stderr, "sandbox runtime error:", err)
			os.Exit(1)
		}
		if err := runHTTP(*listen, *authToken, runtime, evaluator); err != nil {
			fmt.Fprintln(os.Stderr, "sandbox http error:", err)
			os.Exit(1)
		}
		return
	}

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
		if err := policy.ValidateToolPolicy(req.Policy, tool, req.Args); err != nil {
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

func buildRuntime(runtimeType, runtimeConfig string, registry *tools.Registry) (sandboxruntime.Runtime, error) {
	switch strings.ToLower(strings.TrimSpace(runtimeType)) {
	case "", "local":
		return sandboxlocal.NewRuntime(registry), nil
	case "microvm":
		cfg, err := sandboxmicrovm.LoadConfig(runtimeConfig)
		if err != nil {
			return nil, err
		}
		return sandboxmicrovm.NewRuntime(cfg), nil
	default:
		return nil, fmt.Errorf("unknown runtime: %s", runtimeType)
	}
}

func registerBuiltinTools(registry *tools.Registry) {
	_ = registry.Register(builtin.NewCalculator())
	_ = registry.Register(builtin.NewFileSystemTool(nil, 0))
	_ = registry.Register(builtin.NewHTTPClientTool(0))
	_ = registry.Register(builtin.NewWebSearchTool(""))
	_ = registry.Register(builtin.NewFetchTool(0))
}
