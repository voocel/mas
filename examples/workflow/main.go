package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/voocel/mas"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	fmt.Println("=== Multi-Agent Workflow Demo ===")

	// Demo 1: Simple sequential workflow
	simpleWorkflowDemo(apiKey)

	//// Demo 2: Parallel execution
	//parallelWorkflowDemo(apiKey)
	//
	//// Demo 3: Tool integration
	//toolWorkflowDemo(apiKey)
	//
	//// Demo 4: Conditional routing
	//conditionalRoutingDemo(apiKey)
	//
	//// Demo 5: Human-in-the-Loop
	//humanInTheLoopDemo(apiKey)
}

// simpleWorkflowDemo shows a basic sequential workflow
func simpleWorkflowDemo(apiKey string) {
	fmt.Println("\n1. Simple Sequential Workflow:")
	customConfig := mas.AgentConfig{
		Name:        "CustomAgent",
		Model:       "gpt-4.1-mini",
		APIKey:      apiKey,
		BaseURL:     os.Getenv("OPENAI_BASE_URL"),
		Temperature: 0.7,
		MaxTokens:   1000,
	}
	// Create agents
	researcher := mas.NewAgentWithConfig(customConfig).
		WithSystemPrompt("You are a researcher. Analyze the given topic briefly.")

	writer := mas.NewAgentWithConfig(customConfig).
		WithSystemPrompt("You are a writer. Create content based on research.")

	// Build workflow with fluent API
	workflow := mas.NewWorkflow().
		AddNode(mas.NewAgentNode("researcher", researcher)).
		AddNode(mas.NewAgentNode("writer", writer)).
		AddEdge("researcher", "writer").
		SetStart("researcher")

	// Execute workflow
	ctx := context.Background()
	initialData := map[string]any{
		"input": "Research artificial intelligence trends",
	}

	result, err := workflow.Execute(ctx, initialData)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Final output: %v\n", result.Get("output"))
}

// parallelWorkflowDemo shows parallel execution
func parallelWorkflowDemo(apiKey string) {
	fmt.Println("\n2. Parallel Workflow:")

	// Create specialized agents
	techAgent := mas.NewAgent("gpt-4.1", apiKey).
		WithSystemPrompt("You are a tech analyst. Provide technical insights.")

	marketAgent := mas.NewAgent("gpt-4.1", apiKey).
		WithSystemPrompt("You are a market analyst. Provide market insights.")

	// Create parallel node
	parallelNode := mas.NewParallelNode("analysis",
		mas.NewAgentNode("tech", techAgent),
		mas.NewAgentNode("market", marketAgent),
	)

	// Build workflow
	workflow := mas.NewWorkflow().
		AddNode(parallelNode).
		SetStart("analysis")

	// Execute
	ctx := context.Background()
	initialData := map[string]any{
		"input": "Analyze AI startup landscape",
	}

	result, err := workflow.Execute(ctx, initialData)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Parallel analysis completed. Last agent: %v\n", result.Get("last_agent"))
}

// toolWorkflowDemo shows tool integration
func toolWorkflowDemo(apiKey string) {
	fmt.Println("\n3. Tool Integration Workflow:")
	fmt.Println("(Tool integration demo skipped - tools package not implemented yet)")
}

// conditionalRoutingDemo shows conditional workflow routing
func conditionalRoutingDemo(apiKey string) {
	fmt.Println("\n4. Conditional Routing Workflow:")

	// Create agents
	classifier := mas.NewAgent("gpt-4.1", apiKey).
		WithSystemPrompt("Classify the input as 'technical' or 'business'. Respond with only one word.")

	techExpert := mas.NewAgent("gpt-4.1", apiKey).
		WithSystemPrompt("You are a technical expert. Provide technical analysis.")

	bizExpert := mas.NewAgent("gpt-4.1", apiKey).
		WithSystemPrompt("You are a business expert. Provide business analysis.")

	// Build workflow with conditional routing
	workflow := mas.NewWorkflow().
		AddNode(mas.NewAgentNode("classifier", classifier)).
		AddNode(mas.NewAgentNode("tech_expert", techExpert)).
		AddNode(mas.NewAgentNode("biz_expert", bizExpert)).
		AddConditionalRoute("classifier",
			func(ctx *mas.WorkflowContext) bool {
				output := ctx.Get("output")
				return output != nil && fmt.Sprintf("%v", output) == "technical"
			},
			"tech_expert", "biz_expert").
		SetStart("classifier")

	// Execute
	ctx := context.Background()
	initialData := map[string]any{
		"input": "Explain machine learning algorithms",
	}

	result, err := workflow.Execute(ctx, initialData)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Conditional routing completed. Routed to: %v\n", result.Get("last_agent"))
}

// humanInTheLoopDemo shows human-in-the-loop workflow
func humanInTheLoopDemo(apiKey string) {
	fmt.Println("\n5. Human-in-the-Loop Workflow:")

	// Create agents
	drafter := mas.NewAgent("gpt-4.1", apiKey).
		WithSystemPrompt("You are a content drafter. Create initial content.")

	finalizer := mas.NewAgent("gpt-4.1", apiKey).
		WithSystemPrompt("You are a content finalizer. Polish the content based on human feedback.")

	// Create human input provider
	humanProvider := mas.NewConsoleInputProvider()

	// Build workflow with human approval
	workflow := mas.NewWorkflow().
		AddNode(mas.NewAgentNode("drafter", drafter)).
		AddNode(mas.NewHumanNode("reviewer", "Please review the content and provide feedback:", humanProvider)).
		AddNode(mas.NewAgentNode("finalizer", finalizer)).
		AddEdge("drafter", "reviewer").
		AddEdge("reviewer", "finalizer").
		SetStart("drafter")

	// Execute
	ctx := context.Background()
	initialData := map[string]any{
		"input": "Write a brief introduction about artificial intelligence",
	}

	result, err := workflow.Execute(ctx, initialData)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Human-in-the-loop workflow completed.\n")
	fmt.Printf("Final content: %v\n", result.Get("output"))
}
