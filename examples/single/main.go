package main

import (
	"context"
	"fmt"
	"os"

	"github.com/voocel/mas/agent"
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
		agent.WithSystemPrompt("You are a friendly assistant who explains and calculates clearly."),
		agent.WithTools(builtin.NewCalculator()),
	)

	r := runner.New(runner.Config{Model: model})

	ctx := context.Background()
	resp, err := r.Run(ctx, ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "Calculate 15 * 8 + 7",
	})
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Println(resp.Content)
}
