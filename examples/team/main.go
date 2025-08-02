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
	// Team collaboration example for MAS framework
	fmt.Println("MAS Framework - Team Collaboration Example")
	fmt.Println("==========================================")

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set OPENAI_API_KEY environment variable")
	}

	// Example 1: Simple team workflow
	fmt.Println("\n1. Simple Team Workflow:")
	simpleTeamWorkflow(apiKey)

	// Example 2: Research and writing team
	fmt.Println("\n2. Research and Writing Team:")
	researchWritingTeam(apiKey)

	// Example 3: Parallel processing team
	fmt.Println("\n3. Parallel Processing Team:")
	parallelProcessingTeam(apiKey)

	// Example 4: Shared memory team
	fmt.Println("\n4. Shared Memory Team:")
	sharedMemoryTeam(apiKey)

	// Example 5: Specialized agents team
	fmt.Println("\n5. Specialized Agents Team:")
	specializedAgentsTeam(apiKey)
}

// simpleTeamWorkflow demonstrates basic team collaboration
func simpleTeamWorkflow(apiKey string) {
	// Create agents
	researcher := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a researcher. Gather and summarize key information on given topics.").
		WithTemperature(0.3)

	writer := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a writer. Create engaging content based on research provided to you.").
		WithTemperature(0.7)

	// Create team
	team := mas.NewTeam().
		Add("researcher", researcher).
		Add("writer", writer).
		WithFlow("researcher", "writer")

	// Execute task
	result, err := team.Execute(context.Background(), 
		"Create content about the benefits of renewable energy")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Team Result:\n%s\n", result)
}

// researchWritingTeam demonstrates a more complex research and writing workflow
func researchWritingTeam(apiKey string) {
	// Create specialized agents
	researcher := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a research specialist. Your job is to gather factual information, statistics, and key points about any given topic. Provide structured, well-organized research findings.").
		WithTools(tools.WebSearch()).
		WithMemory(memory.Conversation(10)).
		WithTemperature(0.2)

	writer := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a content writer. Transform research findings into engaging, well-structured articles. Focus on readability and flow.").
		WithMemory(memory.Conversation(10)).
		WithTemperature(0.6)

	editor := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are an editor. Review content for clarity, grammar, structure, and overall quality. Provide polished, publication-ready text.").
		WithMemory(memory.Conversation(10)).
		WithTemperature(0.3)

	// Create team with sequential workflow
	team := mas.NewTeam().
		Add("researcher", researcher).
		Add("writer", writer).
		Add("editor", editor).
		WithFlow("researcher", "writer", "editor")

	ctx := context.Background()

	// Execute the workflow
	result, err := team.Execute(ctx, "Write an article about artificial intelligence in healthcare")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Final Article:\n%s\n", result)

	// Show team information
	fmt.Printf("\nTeam Information:\n")
	fmt.Printf("- Available Agents: %v\n", team.ListAgents())
}

// parallelProcessingTeam demonstrates parallel processing with multiple agents
func parallelProcessingTeam(apiKey string) {
	// Create agents for different aspects
	technicalAnalyst := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a technical analyst. Focus on technical aspects, implementation details, and technological implications.").
		WithTemperature(0.4)

	marketAnalyst := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a market analyst. Focus on market trends, business implications, and economic impact.").
		WithTemperature(0.4)

	riskAnalyst := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a risk analyst. Identify potential risks, challenges, and mitigation strategies.").
		WithTemperature(0.3)

	// Create team with parallel execution
	team := mas.NewTeam().
		Add("technical", technicalAnalyst).
		Add("market", marketAnalyst).
		Add("risk", riskAnalyst).
		WithFlow("technical", "market", "risk")

	// Note: Parallel execution would need to be implemented in the framework

	result, err := team.Execute(context.Background(), 
		"Analyze the implementation of blockchain technology in supply chain management")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Parallel Analysis Results:\n%s\n", result)
}

// sharedMemoryTeam demonstrates team with shared memory
func sharedMemoryTeam(apiKey string) {
	// Create shared memory
	sharedMem := memory.ThreadSafe(memory.Conversation(20))

	// Create agents with shared memory
	brainstormer := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a creative brainstormer. Generate innovative ideas and solutions.").
		WithMemory(sharedMem).
		WithTemperature(0.8)

	critic := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a critical analyst. Evaluate ideas for feasibility, pros and cons.").
		WithMemory(sharedMem).
		WithTemperature(0.4)

	synthesizer := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a synthesizer. Combine the best ideas and create a cohesive solution.").
		WithMemory(sharedMem).
		WithTemperature(0.5)

	// Create team with shared memory
	team := mas.NewTeam().
		Add("brainstormer", brainstormer).
		Add("critic", critic).
		Add("synthesizer", synthesizer).
		SetSharedMemory(sharedMem).
		WithFlow("brainstormer", "critic", "synthesizer")

	result, err := team.Execute(context.Background(), 
		"How can we reduce plastic waste in urban environments?")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Collaborative Solution:\n%s\n", result)

	// Show shared memory content
	fmt.Printf("\nShared Memory Contents:\n")
	history, _ := sharedMem.GetHistory(context.Background(), 5)
	for i, msg := range history {
		fmt.Printf("%d. [%s]: %s\n", i+1, msg.Role, 
			truncateString(msg.Content, 100))
	}
}

// specializedAgentsTeam demonstrates agents with specific tools and capabilities
func specializedAgentsTeam(apiKey string) {
	// Create specialized agents with different tools
	dataAnalyst := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a data analyst. Work with numbers, calculations, and data processing.").
		WithTools(tools.Calculator(), tools.JSONParser()).
		WithTemperature(0.2)

	webResearcher := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a web researcher. Find and analyze online information.").
		WithTools(tools.WebSearch(), tools.HTTPRequest(), tools.WebScraper()).
		WithTemperature(0.4)

	documentManager := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a document manager. Handle file operations and document processing.").
		WithTools(tools.FileWriter(), tools.FileReader(), tools.DirectoryLister()).
		WithTemperature(0.3)

	coordinator := mas.NewAgent("gpt-4", apiKey).
		WithSystemPrompt("You are a project coordinator. Integrate findings from specialists into a comprehensive report.").
		WithMemory(memory.Conversation(15)).
		WithTemperature(0.5)

	// Create team
	team := mas.NewTeam().
		Add("data_analyst", dataAnalyst).
		Add("web_researcher", webResearcher).
		Add("document_manager", documentManager).
		Add("coordinator", coordinator).
		WithFlow("data_analyst", "web_researcher", "document_manager", "coordinator")

	result, err := team.Execute(context.Background(), 
		"Research the current state of electric vehicle adoption, analyze the growth numbers, and create a comprehensive report saved to a file")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Specialized Team Result:\n%s\n", result)
}

// dynamicTeamExample demonstrates dynamic team composition
func dynamicTeamExample(apiKey string) {
	fmt.Println("\n6. Dynamic Team Example:")

	// Create a base team
	baseTeam := mas.NewTeam()

	// Add agents dynamically based on task requirements
	if needsResearch := true; needsResearch {
		researcher := mas.NewAgent("gpt-4", apiKey).
			WithSystemPrompt("You are a researcher.")
		baseTeam = baseTeam.Add("researcher", researcher)
	}

	if needsWriting := true; needsWriting {
		writer := mas.NewAgent("gpt-4", apiKey).
			WithSystemPrompt("You are a writer.")
		baseTeam = baseTeam.Add("writer", writer)
	}

	// Configure flow dynamically
	finalTeam := baseTeam.WithFlow("researcher", "writer")

	result, err := finalTeam.Execute(context.Background(), 
		"Create a brief overview of quantum computing")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Dynamic Team Result:\n%s\n", result)
}

// errorHandlingTeam demonstrates error handling in team workflows
func errorHandlingTeam(apiKey string) {
	fmt.Println("\n7. Error Handling in Teams:")

	// Create a team with an agent that might fail
	unreliableAgent := mas.NewAgent("gpt-4", "invalid-key") // Invalid API key
	reliableAgent := mas.NewAgent("gpt-4", apiKey)

	team := mas.NewTeam().
		Add("unreliable", unreliableAgent).
		Add("reliable", reliableAgent).
		WithFlow("unreliable", "reliable")

	result, err := team.Execute(context.Background(), "Test message")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Printf("Unexpected success: %s\n", result)
	}
}

// Helper functions

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}