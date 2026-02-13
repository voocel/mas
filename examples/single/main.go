package main

import (
	"fmt"
	"os"

	"github.com/voocel/agentcore"
	"github.com/voocel/agentcore/llm"
	"github.com/voocel/agentcore/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY not set")
		os.Exit(1)
	}

	model := llm.NewOpenAIModel("gpt-4.1-mini", apiKey)

	agent := agentcore.NewAgent(
		agentcore.WithModel(model),
		agentcore.WithSystemPrompt("You are a helpful coding assistant. Use the provided tools to help users."),
		agentcore.WithTools(
			tools.NewRead(),
			tools.NewWrite(),
			tools.NewEdit(),
			tools.NewBash("."),
		),
		agentcore.WithMaxTurns(20),
	)

	// Subscribe to events for output
	agent.Subscribe(func(ev agentcore.Event) {
		switch ev.Type {
		case agentcore.EventMessageEnd:
			if msg, ok := ev.Message.(agentcore.Message); ok && msg.Role == agentcore.RoleAssistant {
				fmt.Printf("\nAssistant: %s\n", msg.TextContent())
			}
		case agentcore.EventToolExecStart:
			fmt.Printf("  [tool] %s(%v)\n", ev.Tool, string(ev.Args.([]byte)))
		case agentcore.EventToolExecEnd:
			if ev.IsError {
				fmt.Printf("  [tool] %s error\n", ev.Tool)
			}
		case agentcore.EventError:
			fmt.Fprintf(os.Stderr, "Error: %v\n", ev.Err)
		}
	})

	if err := agent.Prompt("List the files in the current directory and tell me what you see."); err != nil {
		fmt.Fprintf(os.Stderr, "prompt error: %v\n", err)
		os.Exit(1)
	}

	agent.WaitForIdle()
}
