package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
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

	modelName := os.Getenv("MODEL")
	if modelName == "" {
		modelName = "gpt-4.1-mini"
	}

	model := llm.NewOpenAIModel(modelName, apiKey)

	agent := mas.NewAgent(
		mas.WithModel(model),
		mas.WithSystemPrompt("You are a helpful coding assistant. Use the provided tools to help users with software engineering tasks."),
		mas.WithTools(
			tools.NewRead(),
			tools.NewWrite(),
			tools.NewEdit(),
			tools.NewBash("."),
		),
		mas.WithMaxTurns(20),
	)

	m := newModel(agent, modelName)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Bridge: agent events -> bubbletea Elm loop
	agent.Subscribe(func(ev mas.Event) {
		p.Send(agentEventMsg{event: ev})
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
