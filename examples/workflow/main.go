package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/voocel/mas"
	"github.com/voocel/mas/tools"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	fmt.Println("=== Multi-Agent Workflow Demo ===")
	
	// Demo 1: Simple sequential workflow
	simpleWorkflowDemo(apiKey)
	
	// Demo 2: Parallel execution
	parallelWorkflowDemo(apiKey)
	
	// Demo 3: Tool integration
	toolWorkflowDemo(apiKey)
}

// simpleWorkflowDemo shows a basic sequential workflow
func simpleWorkflowDemo(apiKey string) {
	fmt.Println("\n1. Simple Sequential Workflow:")
	
	// Create agents
	researcher := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a researcher. Analyze the given topic briefly.")
	
	writer := mas.NewAgent("gpt-4", apiKey).
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
	techAgent := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a tech analyst. Provide technical insights.")
	
	marketAgent := mas.NewAgent("gpt-4", apiKey).
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
	
	// Create agent and tool nodes
	planner := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a planner. Create a calculation plan.")
	
	calculator := mas.NewToolNode("calc", tools.Calculator()).
		WithParams(map[string]any{
			"operation": "multiply",
			"a":         123,
			"b":         456,
		})
	
	reporter := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a reporter. Summarize the calculation result.")
	
	// Build workflow
	workflow := mas.NewWorkflow().
		AddNode(mas.NewAgentNode("planner", planner)).
		AddNode(calculator).
		AddNode(mas.NewAgentNode("reporter", reporter)).
		AddEdge("planner", "calc").
		AddEdge("calc", "reporter").
		SetStart("planner")
	
	// Execute
	ctx := context.Background()
	initialData := map[string]any{
		"input": "Calculate 123 * 456 and explain the result",
	}
	
	result, err := workflow.Execute(ctx, initialData)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	
	fmt.Printf("Tool workflow completed. Result: %v\n", result.Get("tool_result"))
	fmt.Printf("Final report: %v\n", result.Get("output"))
}
