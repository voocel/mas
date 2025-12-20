package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/middleware"
	"github.com/voocel/mas/observer"
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
		agent.WithSystemPrompt("You are a friendly assistant."),
		agent.WithTools(builtin.NewCalculator()),
	)

	metrics := &middleware.MetricsObserver{}
	obs := observer.NewCompositeObserver(
		observer.NewJSONObserver(os.Stdout),
		observer.NewLoggerObserver(os.Stdout),
		metrics,
	)

	r := runner.New(runner.Config{
		Model:    model,
		Observer: obs,
		Tracer:   observer.NewSimpleTimerTracer(os.Stdout),
		Middlewares: []runner.Middleware{
			&middleware.TimeoutMiddleware{LLMTimeout: 10 * time.Second, ToolTimeout: 20 * time.Second},
		},
	})

	resp, err := r.Run(context.Background(), ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "Calculate 12 * 7 + 5",
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("response:", resp.Content)
	fmt.Printf("metrics: %+v\n", metrics.Snapshot())
}
