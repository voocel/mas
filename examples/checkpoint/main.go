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
	fmt.Println("MAS Framework - Checkpoint & Recovery Example")
	fmt.Println("===========================================")

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// Example 1: Basic checkpoint usage
	fmt.Println("\n1. Basic Checkpoint Usage:")
	basicCheckpointExample(apiKey)

	// Example 2: Manual checkpoint creation
	fmt.Println("\n2. Manual Checkpoint Creation:")
	manualCheckpointExample(apiKey)
}

// basicCheckpointExample demonstrates basic checkpoint functionality
func basicCheckpointExample(apiKey string) {
	// Create a checkpointer
	checkpointer := mas.NewMemoryCheckpointer()
	defer checkpointer.Close()

	// Create agents
	researcher := mas.NewAgent("gpt-4o-mini", apiKey).
		WithSystemPrompt("You are a researcher. Analyze the given topic.")

	writer := mas.NewAgent("gpt-4o-mini", apiKey).
		WithSystemPrompt("You are a writer. Create content based on research.")

	// Build workflow with checkpointer
	workflow := mas.NewWorkflow().
		AddNode(mas.NewAgentNode("researcher", researcher)).
		AddNode(mas.NewAgentNode("writer", writer)).
		AddEdge("researcher", "writer").
		SetStart("researcher").
		WithCheckpointer(checkpointer)

	// Execute workflow with checkpointing
	ctx := context.Background()
	initialData := map[string]any{
		"input": "Artificial Intelligence trends in 2024",
	}

	result, err := workflow.ExecuteWithCheckpoint(ctx, initialData)
	if err != nil {
		log.Printf("Workflow execution failed: %v", err)
		return
	}

	fmt.Printf("Workflow completed successfully!\n")
	fmt.Printf("Final output: %v\n", result.Get("output"))

	// List checkpoints
	fmt.Println("\nCheckpoints created during execution:")
	infos, err := checkpointer.List(ctx, result.ID)
	if err != nil {
		log.Printf("Failed to list checkpoints: %v", err)
		return
	}

	for _, info := range infos {
		fmt.Printf("- ID: %s, Node: %s, Time: %s\n", 
			info.ID, info.CurrentNode, info.Timestamp.Format(time.RFC3339))
	}
}

// manualCheckpointExample demonstrates manual checkpoint creation and recovery
func manualCheckpointExample(apiKey string) {
	checkpointer := mas.NewMemoryCheckpointer()
	defer checkpointer.Close()

	// Create a simple agent
	agent := mas.NewAgent("gpt-4o-mini", apiKey).
		WithSystemPrompt("You are a helpful assistant.")

	// Create workflow
	workflow := mas.NewWorkflow().
		AddNode(mas.NewAgentNode("step1", agent)).
		AddNode(mas.NewAgentNode("step2", agent)).
		AddEdge("step1", "step2").
		SetStart("step1").
		WithCheckpointer(checkpointer)

	ctx := context.Background()
	initialData := map[string]any{
		"input": "Process this task step by step",
	}

	// Execute workflow
	result, err := workflow.ExecuteWithCheckpoint(ctx, initialData)
	if err != nil {
		log.Printf("Workflow execution failed: %v", err)
		return
	}

	fmt.Printf("Workflow %s completed successfully\n", result.ID)

	// Create manual checkpoint
	fmt.Println("Creating manual checkpoint...")
	manualCheckpoint := mas.CreateCheckpoint(
		result.ID, 
		"step2", 
		[]string{"step1"}, 
		result, 
		mas.CheckpointTypeManual,
	)
	
	err = checkpointer.Save(ctx, manualCheckpoint)
	if err != nil {
		log.Printf("Failed to save manual checkpoint: %v", err)
		return
	}
	
	fmt.Printf("Manual checkpoint created: %s\n", manualCheckpoint.ID)

	// Show all checkpoints
	infos, _ := checkpointer.List(ctx, result.ID)
	fmt.Printf("Total checkpoints: %d\n", len(infos))
	for i, info := range infos {
		fmt.Printf("%d. %s (Type: %s, Node: %s)\n", 
			i+1, info.ID, info.Type, info.CurrentNode)
	}

	// Cleanup demonstration
	fmt.Println("Demonstrating checkpoint cleanup...")
	err = checkpointer.Cleanup(ctx, 0)
	if err != nil {
		log.Printf("Cleanup failed: %v", err)
		return
	}
	
	fmt.Println("Checkpoint cleanup completed")
}