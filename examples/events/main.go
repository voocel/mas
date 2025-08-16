package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	fmt.Println("=== MAS Unified Event System Demo ===")

	// Demo 1: Simple agent without events
	simpleAgentDemo(apiKey)

	// Demo 2: Simple agent with events
	agentWithEventsDemo(apiKey)

	// Demo 3: Workflow with events
	workflowWithEventsDemo(apiKey)

	// Demo 4: Event streaming
	eventStreamingDemo(apiKey)
}

// simpleAgentDemo shows agent without events (existing API unchanged)
func simpleAgentDemo(apiKey string) {
	fmt.Println("\n1. Simple Agent (No Events):")

	// Existing API unchanged
	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a helpful assistant.")

	response, err := agent.Chat(context.Background(), "Hello!")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}

// agentWithEventsDemo shows agent with optional events
func agentWithEventsDemo(apiKey string) {
	fmt.Println("\n2. Agent with Events (Optional Enhancement):")

	eventBus := mas.NewEventBus()

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a helpful assistant.").
		WithEventBus(eventBus)

	// Subscribe to events
	eventBus.Subscribe(mas.EventAgentChatStart, func(ctx context.Context, event mas.Event) error {
		fmt.Printf("Chat started: %s\n", event.Data["message"])
		return nil
	})

	eventBus.Subscribe(mas.EventAgentChatEnd, func(ctx context.Context, event mas.Event) error {
		fmt.Printf("Chat completed: %s\n", event.Data["response"])
		return nil
	})

	// Same Chat API
	response, err := agent.Chat(context.Background(), "Hello! How are you?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Final response: %s\n", response)
}

// workflowWithEventsDemo shows workflow with optional events
func workflowWithEventsDemo(apiKey string) {
	fmt.Println("\n3. Workflow with Events:")

	// Create shared event bus
	eventBus := mas.NewEventBus()

	// Create agents (can optionally have events)
	researcher := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a researcher. Provide brief research.").
		WithEventBus(eventBus)

	writer := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a writer. Create content based on research.").
		WithEventBus(eventBus)

	workflow := mas.NewWorkflow().
		AddNode(mas.NewAgentNode("researcher", researcher)).
		AddNode(mas.NewAgentNode("writer", writer)).
		AddEdge("researcher", "writer").
		SetStart("researcher").
		WithEventBus(eventBus)

	// Subscribe to workflow events
	eventBus.Subscribe(mas.EventWorkflowStart, func(ctx context.Context, event mas.Event) error {
		fmt.Printf("Workflow started\n")
		return nil
	})

	eventBus.Subscribe(mas.EventNodeStart, func(ctx context.Context, event mas.Event) error {
		fmt.Printf("📍 Node started: %s\n", event.Data["node_id"])
		return nil
	})

	eventBus.Subscribe(mas.EventWorkflowEnd, func(ctx context.Context, event mas.Event) error {
		fmt.Printf("Workflow completed\n")
		return nil
	})

	// Same Execute API
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

// eventStreamingDemo shows real-time event streaming
func eventStreamingDemo(apiKey string) {
	fmt.Println("\n4. Event Streaming:")

	eventBus := mas.NewEventBus()

	agent := mas.NewAgent("gpt-4.1-mini", apiKey).
		WithSystemPrompt("You are a helpful assistant.").
		WithEventBus(eventBus)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eventStream, err := agent.StreamEvents(ctx, mas.EventAgentChatStart, mas.EventAgentChatEnd)
	if err != nil {
		log.Printf("Error starting event stream: %v", err)
		return
	}

	// Process events in background
	go func() {
		for event := range eventStream {
			switch event.Type {
			case mas.EventAgentChatStart:
				fmt.Printf("Streaming: Chat started: %s\n", event.Data["message"])
			case mas.EventAgentChatEnd:
				fmt.Printf("Streaming: Chat ended: %s\n", event.Data["response"])
			}
		}
	}()

	// Multiple conversations
	conversations := []string{
		"What is AI?",
		"How does ML work?",
		"Tell me about neural networks.",
	}

	for i, message := range conversations {
		fmt.Printf("\n--- Conversation %d ---\n", i+1)
		response, err := agent.Chat(ctx, message)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		fmt.Printf("Response: %s\n", response)
		time.Sleep(1 * time.Second)
	}

	time.Sleep(2 * time.Second) // Wait for events
}
