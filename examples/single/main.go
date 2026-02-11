package main

import (
	"fmt"
	"os"

	"github.com/voocel/mas"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY not set")
		os.Exit(1)
	}

	model := llm.NewOpenAIModel("gpt-4.1-mini", apiKey)

	agent := mas.NewAgent(
		mas.WithModel(model),
		mas.WithSystemPrompt("You are a helpful coding assistant. Use the provided tools to help users."),
		mas.WithTools(
			tools.NewRead(),
			tools.NewWrite(),
			tools.NewEdit(),
			tools.NewBash("."),
		),
		mas.WithMaxTurns(20),
	)

	// Subscribe to events for output
	agent.Subscribe(func(ev mas.Event) {
		switch ev.Type {
		case mas.EventMessageEnd:
			if msg, ok := ev.Message.(mas.Message); ok && msg.Role == mas.RoleAssistant {
				fmt.Printf("\nAssistant: %s\n", msg.TextContent())
			}
		case mas.EventToolExecStart:
			fmt.Printf("  [tool] %s(%v)\n", ev.Tool, string(ev.Args.([]byte)))
		case mas.EventToolExecEnd:
			if ev.IsError {
				fmt.Printf("  [tool] %s error\n", ev.Tool)
			}
		case mas.EventError:
			fmt.Fprintf(os.Stderr, "Error: %v\n", ev.Err)
		}
	})

	if err := agent.Prompt("List the files in the current directory and tell me what you see."); err != nil {
		fmt.Fprintf(os.Stderr, "prompt error: %v\n", err)
		os.Exit(1)
	}

	agent.WaitForIdle()
}
