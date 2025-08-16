package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/voocel/mas"
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

	// Example 1: Custom tool creation
	fmt.Println("\n1. Custom Tool Example:")
	customToolExample(apiKey)

	// Example 2: Tool registry
	fmt.Println("\n2. Multiple Tools Example:")
	toolRegistryExample(apiKey)
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

// toolRegistryExample demonstrates using multiple tools together
func toolRegistryExample(apiKey string) {
	// Create multiple tools directly
	randomTool := createRandomNumberTool()
	greetingTool := createGreetingTool()
	
	// List all tools
	fmt.Println("Available tools:")
	tools := []mas.Tool{randomTool, greetingTool}
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
	}

	// Create agent with tools directly
	agent := mas.NewAgent("gpt-4", apiKey).
		WithTools(tools...).
		WithSystemPrompt("You have access to multiple tools. Use them as needed.")

	response, err := agent.Chat(context.Background(),
		"What tools do you have available? Use one of them.")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// createGreetingTool creates a simple greeting tool
func createGreetingTool() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"name": mas.StringProperty("Name to greet"),
		},
		Required: []string{"name"},
	}

	return mas.NewTool(
		"greeting",
		"Generates a personalized greeting message",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			name, ok := params["name"].(string)
			if !ok {
				return nil, fmt.Errorf("name must be a string")
			}

			return map[string]interface{}{
				"greeting": fmt.Sprintf("Hello, %s! Welcome to the MAS framework!", name),
			}, nil
		},
	)
}
