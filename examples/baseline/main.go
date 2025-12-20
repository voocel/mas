package main

import (
	"context"
	"fmt"
	"os"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/middleware"
	"github.com/voocel/mas/observer"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
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

	checkpointer := runner.NewMemoryCheckpointer()
	policy := middleware.NewToolAccessPolicy(
		[]string{"calculator"},
		[]tools.Capability{tools.CapabilityNetwork},
	)
	hitl := &middleware.HITLMiddleware{
		Approver: middleware.HITLFunc{
			LLM: func(ctx context.Context, state *runner.State, req *llm.Request) middleware.HITLDecision {
				fmt.Printf("approve llm run=%s step=%s\n", state.RunID, state.StepID)
				return middleware.Allow()
			},
			Tool: func(ctx context.Context, state *runner.ToolState) middleware.HITLDecision {
				if state.Call != nil {
					fmt.Printf("approve tool=%s run=%s step=%s\n", state.Call.Name, state.RunID, state.StepID)
				}
				return middleware.Allow()
			},
		},
	}

	r := runner.New(runner.Config{
		Model:          model,
		Checkpointer:   checkpointer,
		RunIDGenerator: func() string { return "run_baseline_demo" },
		Middlewares:    []runner.Middleware{policy, hitl},
		Observer:       observer.NewLoggerObserver(os.Stdout),
		Tracer:         observer.NewSimpleTimerTracer(os.Stdout),
	})

	ctx := context.Background()
	result, err := r.RunWithResult(ctx, ag, schema.Message{
		Role:    schema.RoleUser,
		Content: "Calculate 12 * 7",
	})
	if err != nil {
		fmt.Printf("run error: %v\n", err)
		return
	}
	fmt.Println("first:", result.Message.Content)

	ckpt, err := checkpointer.Load(ctx, "run_baseline_demo")
	if err != nil {
		fmt.Printf("load checkpoint error: %v\n", err)
		return
	}

	r2 := runner.New(runner.Config{
		Model:       model,
		Middlewares: []runner.Middleware{policy, hitl},
	})
	resumed, err := r2.RunFromCheckpoint(ctx, ag, ckpt)
	if err != nil {
		fmt.Printf("resume error: %v\n", err)
		return
	}
	fmt.Println("resumed:", resumed.Message.Content)
}
