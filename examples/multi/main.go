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

	mainModel := llm.NewOpenAIModel("gpt-4.1-mini", apiKey)
	scoutModel := llm.NewOpenAIModel("gpt-4.1-mini", apiKey)

	// Define sub-agent configurations (like pi's .md agent files)
	scout := mas.SubAgentConfig{
		Name:         "scout",
		Description:  "Fast codebase reconnaissance",
		Model:        scoutModel,
		SystemPrompt: "You are a scout agent. Quickly explore the codebase and report what you find. Be concise.",
		Tools: []mas.Tool{
			tools.NewRead(),
			tools.NewBash("."),
		},
		MaxTurns: 5,
	}

	reviewer := mas.SubAgentConfig{
		Name:         "reviewer",
		Description:  "Code review specialist",
		Model:        mainModel,
		SystemPrompt: "You are a code reviewer. Review the code and provide constructive feedback on quality, style, and correctness.",
		Tools: []mas.Tool{
			tools.NewRead(),
			tools.NewBash("."),
		},
		MaxTurns: 5,
	}

	// Main agent has the subagent tool — it delegates to scout/reviewer
	agent := mas.NewAgent(
		mas.WithModel(mainModel),
		mas.WithSystemPrompt(
			"You are a coding assistant. Use the subagent tool to delegate tasks:\n"+
				"- Use 'scout' for codebase exploration\n"+
				"- Use 'reviewer' for code review\n"+
				"You can use chain mode to scout first, then review based on findings.",
		),
		mas.WithTools(
			tools.NewRead(),
			tools.NewWrite(),
			tools.NewEdit(),
			tools.NewBash("."),
			mas.NewSubAgentTool(scout, reviewer),
		),
		mas.WithMaxTurns(20),
	)

	agent.Subscribe(func(ev mas.Event) {
		switch ev.Type {
		case mas.EventMessageEnd:
			if msg, ok := ev.Message.(mas.Message); ok && msg.Role == mas.RoleAssistant {
				fmt.Printf("\nAssistant: %s\n", msg.TextContent())
			}
		case mas.EventToolExecStart:
			fmt.Printf("  [tool] %s\n", ev.Tool)
		case mas.EventError:
			fmt.Fprintf(os.Stderr, "Error: %v\n", ev.Err)
		}
	})

	if err := agent.Prompt("Explore the current directory structure, then review any Go files you find. Use chain mode: scout first, then review."); err != nil {
		fmt.Fprintf(os.Stderr, "prompt error: %v\n", err)
		os.Exit(1)
	}

	agent.WaitForIdle()
}
