package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/executor"
	"github.com/voocel/mas/executor/local"
	"github.com/voocel/mas/executor/sandbox"
	sandboxclient "github.com/voocel/mas/executor/sandbox/client"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools/builtin"
)

func main() {
	model := llm.NewOpenAIModel(
		"gpt-5",
		os.Getenv("OPENAI_API_KEY"),
		os.Getenv("OPENAI_API_BASE_URL"),
	)

	ag := agent.New(
		"assistant",
		"assistant",
		agent.WithSystemPrompt("You are a helpful assistant."),
		agent.WithTools(builtin.NewCalculator()),
	)

	exec := buildExecutor()
	defer exec.Close()

	r := runner.New(runner.Config{
		Model:        model,
		ToolExecutor: exec,
		ExecutorPolicy: executor.Policy{
			AllowedTools: []string{"calculator"},
		},
	})

	resp, err := r.Run(context.Background(), ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "Compute 15 * 8 + 7",
	})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Println(resp.Content)
}

func buildExecutor() executor.ToolExecutor {
	// Set MAS_SANDBOX_MODE=http to use the HTTP control plane.
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("MAS_SANDBOX_MODE")))
	if mode == "http" {
		endpoint := os.Getenv("MAS_SANDBOX_URL")
		if endpoint == "" {
			endpoint = "http://127.0.0.1:8080"
		}
		fmt.Printf("sandbox executor: http (%s)\n", endpoint)
		client := sandboxclient.NewHTTPClient(endpoint)
		client.AuthToken = os.Getenv("MAS_SANDBOX_TOKEN")
		return sandbox.NewSandboxExecutor(client)
	}

	path := os.Getenv("MAS_SANDBOXD")
	if path == "" {
		path = "mas-sandboxd"
	}
	fmt.Printf("sandbox executor: local (%s)\n", path)
	return local.NewLocalExecutor(path)
}
