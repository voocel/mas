package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/voocel/mas"
	"github.com/voocel/mas/memory"
)

func main() {
	// Basic usage example for MAS framework
	fmt.Println("MAS Framework - Basic Usage Example")
	fmt.Println("===================================")

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// Example 1: Simple chat
	fmt.Println("\n1. Simple Chat Example:")
	simpleChat(apiKey)

	// Example 2: Chat with memory
	fmt.Println("\n2. Chat with Memory Example:")
	chatWithMemory(apiKey)

	// Example 3: Chat with system prompt
	fmt.Println("\n3. Chat with System Prompt Example:")
	chatWithSystemPrompt(apiKey)

	// Example 4: Configuration options
	fmt.Println("\n4. Configuration Options Example:")
	configurationOptions(apiKey)
}

// simpleChat demonstrates basic agent usage
func simpleChat(apiKey string) {
	// Create a simple agent
	agent := mas.NewAgent("gpt-4", apiKey)

	// Chat with the agent
	response, err := agent.Chat(context.Background(), "Hello! Please introduce yourself.")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Agent: %s\n", response)
}

// chatWithMemory demonstrates agent with conversation memory
func chatWithMemory(apiKey string) {
	// Create agent with conversation memory
	agent := mas.NewAgent("gpt-4", apiKey).
		WithMemory(memory.Conversation(10)) // Remember last 10 messages

	ctx := context.Background()

	// First message
	response1, err := agent.Chat(ctx, "My name is Alice. What's your name?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("Agent: %s\n", response1)

	// Second message - agent should remember the name
	response2, err := agent.Chat(ctx, "What's my name?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("Agent: %s\n", response2)
}

// chatWithSystemPrompt demonstrates agent with custom system prompt
func chatWithSystemPrompt(apiKey string) {
	// Create agent with custom system prompt
	agent := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a helpful math tutor. Always explain your reasoning step by step.").
		WithTemperature(0.3). // Lower temperature for more consistent responses
		WithMaxTokens(500)

	response, err := agent.Chat(context.Background(), "How do I solve the equation 2x + 5 = 15?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Math Tutor: %s\n", response)
}

// configurationOptions demonstrates various configuration options
func configurationOptions(apiKey string) {
	// Create agent with full configuration
	config := mas.AgentConfig{
		Name:         "CustomAgent",
		Model:        "gpt-4",
		APIKey:       apiKey,
		SystemPrompt: "You are a creative writing assistant.",
		Temperature:  0.8, // Higher temperature for more creativity
		MaxTokens:    300,
		Memory:       memory.Conversation(5),
		State:        make(map[string]interface{}),
	}

	agent := mas.NewAgentWithConfig(config)

	// Set some state
	agent.SetState("genre", "science fiction")
	agent.SetState("tone", "mysterious")

	response, err := agent.Chat(context.Background(), "Write a short opening for a story.")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Creative Assistant: %s\n", response)

	// Show agent info
	fmt.Printf("Agent Name: %s\n", agent.Name())
	fmt.Printf("Agent Model: %s\n", agent.Model())
	fmt.Printf("Genre Setting: %v\n", agent.GetState("genre"))
}

// errorHandling demonstrates error handling patterns
func errorHandling(apiKey string) {
	fmt.Println("\n5. Error Handling Example:")

	// Example with invalid API key
	invalidAgent := mas.NewAgent("gpt-4", "invalid-key")
	_, err := invalidAgent.Chat(context.Background(), "Hello")
	if err != nil {
		fmt.Printf("Expected error with invalid API key: %v\n", err)
	}

	// Example with empty message
	agent := mas.NewAgent("gpt-4", apiKey)
	_, err = agent.Chat(context.Background(), "")
	if err != nil {
		fmt.Printf("Error with empty message: %v\n", err)
	}
}

// performanceExample demonstrates performance considerations
func performanceExample(apiKey string) {
	fmt.Println("\n6. Performance Example:")

	agent := mas.NewAgent("gpt-4", apiKey).
		WithMaxTokens(100). // Limit tokens for faster response
		WithTemperature(0.1) // Lower temperature for more deterministic responses

	start := context.Background()

	response, err := agent.Chat(start, "What is 2+2? Give a brief answer.")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Quick Response: %s\n", response)
}