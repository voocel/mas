package main

import (
	"context"
	"fmt"

	"github.com/voocel/mas"
)

func main() {
	fmt.Println("MAS Framework Verification Test")
	fmt.Println("===============================")

	// Test 1: Basic Agent Creation
	fmt.Println("\nTest 1: Basic Agent Creation")
	agent := mas.NewAgent("gpt-4o", "test-key")
	fmt.Printf("Agent Name: %s\n", agent.Name())
	fmt.Printf("Agent Model: %s\n", agent.Model())

	// Test 2: Agent with Configuration
	fmt.Println("\nTest 2: Agent with Fluent Configuration")
	configuredAgent := mas.NewAgent("gpt-4.1", "test-key").
		WithSystemPrompt("You are a helpful assistant.").
		WithTemperature(0.7).
		WithMaxTokens(1000).
		WithMemory(mas.NewConversationMemory(10))

	fmt.Printf("Configured Agent: %s\n", configuredAgent.Name())

	// Test 3: Custom Tools
	fmt.Println("\nTest 3: Tool Creation")
	greetingTool := mas.NewSimpleTool("greeting", "Generate a greeting message", func(ctx context.Context, params map[string]any) (any, error) {
		return "Hello, World!", nil
	})

	fmt.Printf("Greeting Tool: %s - %s\n", greetingTool.Name(), greetingTool.Description())

	// Test 4: Agent with Tools
	fmt.Println("\nTest 4: Agent with Tools")
	toolAgent := mas.NewAgent("gpt-4.1", "test-key").
		WithTools(greetingTool).
		WithMemory(mas.NewConversationMemory(5))

	fmt.Printf("Agent with tools created successfully: %s\n", toolAgent.Name())

	// Test 5: Memory Systems
	fmt.Println("\nTest 5: Memory Systems")
	convMemory := mas.NewConversationMemory(10)
	summaryMemory := mas.NewSummaryMemory(20)

	fmt.Printf("Conversation Memory: %T\n", convMemory)
	fmt.Printf("Summary Memory: %T\n", summaryMemory)

	// Test 6: Workflow
	fmt.Println("\nTest 6: Workflow")
	researcher := mas.NewAgent("gpt-4.1", "test-key").
		WithSystemPrompt("You are a researcher.")
	writer := mas.NewAgent("gpt-4.1", "test-key").
		WithSystemPrompt("You are a writer.")

	_ = mas.NewWorkflow().
		AddNode(mas.NewAgentNode("researcher", researcher)).
		AddNode(mas.NewAgentNode("writer", writer)).
		AddEdge("researcher", "writer").
		SetStart("researcher")

	fmt.Printf("Workflow created with nodes\n")
}
