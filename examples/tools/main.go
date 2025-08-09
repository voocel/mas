package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/voocel/mas"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/tools"
)

func main() {
	// Tools usage example for MAS framework
	fmt.Println("MAS Framework - Tools Usage Example")
	fmt.Println("===================================")

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// Example 1: Calculator tool
	fmt.Println("\n1. Calculator Tool Example:")
	calculatorExample(apiKey)

	// Example 2: HTTP request tool
	fmt.Println("\n2. HTTP Request Tool Example:")
	httpRequestExample(apiKey)

	// Example 3: File operations
	fmt.Println("\n3. File Operations Example:")
	fileOperationsExample(apiKey)

	// Example 4: File sandbox
	fmt.Println("\n4. File Sandbox Example:")
	fileSandboxExample(apiKey)

	// Example 5: Web search and scraping
	fmt.Println("\n5. Web Tools Example:")
	webToolsExample(apiKey)

	// Example 6: Multiple tools
	fmt.Println("\n6. Multiple Tools Example:")
	multipleToolsExample(apiKey)

	// Example 7: Custom tool
	fmt.Println("\n7. Custom Tool Example:")
	customToolExample(apiKey)
}

// calculatorExample demonstrates calculator tool usage
func calculatorExample(apiKey string) {
	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(tools.Calculator()).
		WithSystemPrompt("You are a math assistant. Use the calculator tool for computations.")

	response, err := agent.Chat(context.Background(), "What is 123 multiplied by 456?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// httpRequestExample demonstrates HTTP request tool
func httpRequestExample(apiKey string) {
	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(tools.HTTPRequest(), tools.JSONParser()).
		WithSystemPrompt("You are a web API assistant. Use HTTP tools to fetch and analyze data.")

	response, err := agent.Chat(context.Background(),
		"Make a GET request to https://httpbin.org/json and tell me about the response")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// fileOperationsExample demonstrates file operation tools
func fileOperationsExample(apiKey string) {
	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(
			tools.FileWriter(),
			tools.FileReader(),
			tools.FileInfo(),
			tools.DirectoryLister(),
		).
		WithSystemPrompt("You are a file management assistant. Help users with file operations.")

	ctx := context.Background()

	// Create a test file
	response1, err := agent.Chat(ctx,
		"Create a file called 'test.txt' with the content 'Hello, MAS Framework!'")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("Agent: %s\n", response1)

	// Read the file back
	response2, err := agent.Chat(ctx, "Read the contents of 'test.txt'")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("Agent: %s\n", response2)

	// Get file info
	response3, err := agent.Chat(ctx, "Get information about 'test.txt'")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("Agent: %s\n", response3)
}

// fileSandboxExample demonstrates file tools with sandbox restrictions
func fileSandboxExample(apiKey string) {
	ctx := context.Background()

	// Example 1: No sandbox (unrestricted)
	fmt.Println("  Unrestricted access:")
	agent1 := mas.NewAgent("gpt-4", apiKey).
		WithTools(tools.FileReader()).
		WithSystemPrompt("You are a file assistant.")

	response1, err := agent1.Chat(ctx, "List current directory")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("  Agent: %s\n", response1)
	}

	// Example 2: Current directory only
	fmt.Println("  Current directory only:")
	sandbox := tools.DefaultSandbox()
	agent2 := mas.NewAgent("gpt-4", apiKey).
		WithTools(tools.DirectoryListerWithSandbox(sandbox)).
		WithSystemPrompt("You are restricted to current directory.")

	response2, err := agent2.Chat(ctx, "Try to list parent directory '../'")
	if err != nil {
		fmt.Printf("  Expected restriction: %v\n", err)
	} else {
		fmt.Printf("  Agent: %s\n", response2)
	}

	// Example 3: Custom allowed paths
	fmt.Println("  Custom allowed paths:")
	customSandbox := &tools.FileSandbox{
		AllowedPaths:    []string{"./examples"},
		AllowCurrentDir: false,
	}
	agent3 := mas.NewAgent("gpt-4", apiKey).
		WithTools(tools.DirectoryListerWithSandbox(customSandbox)).
		WithSystemPrompt("You can only access ./examples directory.")

	response3, err := agent3.Chat(ctx, "List ./examples directory")
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("  Agent: %s\n", response3)
	}
}

// webToolsExample demonstrates web search and scraping tools
func webToolsExample(apiKey string) {
	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(
			tools.WebSearch(),
			tools.WebScraper(),
			tools.DomainInfo(),
		).
		WithSystemPrompt("You are a web research assistant. Help users find and analyze web information.")

	response, err := agent.Chat(context.Background(),
		"Search for information about 'artificial intelligence' and tell me what you find")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// multipleToolsExample demonstrates using multiple tools together
func multipleToolsExample(apiKey string) {
	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(
			tools.Calculator(),
			tools.HTTPRequest(),
			tools.FileWriter(),
			tools.WebSearch(),
		).
		WithMemory(memory.Conversation(20)).
		WithSystemPrompt("You are a versatile assistant with access to multiple tools. Use them as needed to help the user.")

	ctx := context.Background()

	// Complex task requiring multiple tools
	response, err := agent.Chat(ctx,
		"Calculate 15% of 250, then search for information about percentage calculations, and save the result to a file called 'calculation_result.txt'")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// customToolExample demonstrates creating and using custom tools
func customToolExample(apiKey string) {
	// Create a custom tool for generating random numbers
	randomTool := createRandomNumberTool()

	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(randomTool).
		WithSystemPrompt("You are an assistant with a random number generator tool.")

	response, err := agent.Chat(context.Background(),
		"Generate a random number between 1 and 100")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// createRandomNumberTool creates a custom random number generator tool
func createRandomNumberTool() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"min": mas.NumberProperty("Minimum value (inclusive)"),
			"max": mas.NumberProperty("Maximum value (inclusive)"),
		},
		Required: []string{"min", "max"},
	}

	return mas.NewTool(
		"random_number",
		"Generates a random number between min and max values",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			min, ok1 := params["min"].(float64)
			max, ok2 := params["max"].(float64)

			if !ok1 || !ok2 {
				return nil, fmt.Errorf("min and max must be numbers")
			}

			if min > max {
				return nil, fmt.Errorf("min cannot be greater than max")
			}

			// Simple random number generation (not cryptographically secure)
			range_ := max - min + 1
			// Using a simple hash-based approach for demonstration
			hash := int64(min*31+max*17) % 1000
			if hash < 0 {
				hash = -hash
			}

			result := min + float64(hash%int64(range_))

			return map[string]interface{}{
				"min":    min,
				"max":    max,
				"result": result,
			}, nil
		},
	)
}

// toolRegistryExample demonstrates using tool registry for managing tools
func toolRegistryExample(apiKey string) {
	fmt.Println("\n7. Tool Registry Example:")

	// Create a tool registry
	registry := mas.NewToolRegistry()

	// Register tools
	registry.Register(tools.Calculator())
	registry.Register(tools.HTTPRequest())
	registry.Register(createRandomNumberTool())

	// List all tools
	fmt.Println("Available tools:")
	for _, name := range registry.Names() {
		tool, _ := registry.Get(name)
		fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
	}

	// Create agent with tools from registry
	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(registry.List()...).
		WithSystemPrompt("You have access to multiple tools. Use them as needed.")

	response, err := agent.Chat(context.Background(),
		"What tools do you have available?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// toolErrorHandlingExample demonstrates error handling with tools
func toolErrorHandlingExample(apiKey string) {
	fmt.Println("\n8. Tool Error Handling Example:")

	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(tools.Calculator()).
		WithSystemPrompt("You are a math assistant. Handle errors gracefully.")

	// This should cause a division by zero error
	response, err := agent.Chat(context.Background(), "What is 10 divided by 0?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}
