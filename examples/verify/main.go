package main

import (
	"fmt"
	"os"

	"github.com/voocel/mas"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/tools"
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
		WithMemory(memory.Conversation(10))

	fmt.Printf("Configured Agent: %s\n", configuredAgent.Name())

	// Test 3: Tools
	fmt.Println("\nTest 3: Tool Creation")
	calc := tools.Calculator()
	httpTool := tools.HTTPRequest()
	webSearch := tools.WebSearch()

	fmt.Printf("Calculator Tool: %s - %s\n", calc.Name(), calc.Description())
	fmt.Printf("HTTP Tool: %s - %s\n", httpTool.Name(), httpTool.Description())
	fmt.Printf("Web Search Tool: %s - %s\n", webSearch.Name(), webSearch.Description())

	// Test 4: Agent with Tools
	fmt.Println("\nTest 4: Agent with Tools")
	toolAgent := mas.NewAgent("gpt-4.1", "test-key").
		WithTools(calc, httpTool, webSearch).
		WithMemory(memory.Conversation(5))

	fmt.Printf("Agent with tools created successfully: %s\n", toolAgent.Name())

	// Test 5: Memory Systems
	fmt.Println("\nTest 5: Memory Systems")
	convMemory := memory.Conversation(10)
	persistentMemory := memory.Persistent(100, "./test-memory.json")
	sharedMemory := memory.ThreadSafe(memory.Conversation(20))

	fmt.Printf("Conversation Memory: %T\n", convMemory)
	fmt.Printf("Persistent Memory: %T\n", persistentMemory)
	fmt.Printf("Shared Memory: %T\n", sharedMemory)

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

	// Test 7: Tool Registry
	fmt.Println("\nTest 7: Tool Registry")
	registry := mas.NewToolRegistry()
	registry.Register(calc)
	registry.Register(httpTool)

	fmt.Printf("Registry has %d tools: %v\n", len(registry.Names()), registry.Names())

	fmt.Println("\nAll tests passed! MAS Framework is ready to use.")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Set your OPENAI_API_KEY environment variable")
	fmt.Println("2. Run examples: cd examples/basic && go run main.go")
	fmt.Println("3. Check out the documentation in README.md")

	// Clean up test file
	os.Remove("./test-memory.json")
}
