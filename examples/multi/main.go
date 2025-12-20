package main

import (
	"context"
	"fmt"
	"os"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/multi"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

func main() {
	model := llm.NewOpenAIModel(
		"gpt-5",
		os.Getenv("OPENAI_API_KEY"),
		os.Getenv("OPENAI_API_BASE_URL"),
	)

	researcher := agent.New(
		"researcher",
		"researcher",
		agent.WithSystemPrompt("You are a research assistant. Analyze first, then provide the conclusion."),
	)

	writer := agent.New(
		"writer",
		"writer",
		agent.WithSystemPrompt("You are a writing assistant. Produce a structured summary."),
	)

	team := multi.NewTeam()
	_ = team.Add("researcher", researcher)
	_ = team.Add("writer", writer)

	ag, err := team.Route("researcher")
	if err != nil {
		fmt.Println("route error:", err)
		return
	}

	r := runner.New(runner.Config{Model: model})
	resp, err := r.Run(context.Background(), ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "Briefly explain the core concepts of Go's concurrency model.",
	})
	if err != nil {
		fmt.Println("run error:", err)
		return
	}

	fmt.Println(resp.Content)
}
