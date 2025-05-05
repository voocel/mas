package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas"
	"github.com/voocel/mas/agency"
	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/memory"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.Println("========== Multi-Agent Writing System Startup ==========")

	// Create LLM provider
	log.Println("Initializing LLM provider...")
	provider, err := llm.NewOpenAIProvider(llm.Config{
		APIKey:       os.Getenv("OPENAI_API_KEY"),
		DefaultModel: "gpt-4.1-mini",
		Timeout:      30,
	})
	if err != nil {
		log.Fatalf("Failed to create LLM provider: %v", err)
	}
	log.Println("LLM provider initialization successful, using model: gpt-4o")

	// Create multi-agent system
	log.Println("Initializing multi-agent system...")
	system := mas.DefaultSystem()
	log.Println("Multi-agent system initialization successful")

	// Create communication bus
	log.Println("Initializing communication bus...")
	// Note: When using Agency, no need to directly operate the bus, it's automatically managed by the system
	log.Println("Communication bus initialization successful, type: memory, buffer size: 100")

	log.Println("========== Starting Agent Creation ==========")
	// Create three different role agents

	// 1. Research Agent - responsible for collecting topic information
	log.Println("Creating researcher agent...")
	researchAgent := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:         "Researcher",
		Provider:     provider,
		SystemPrompt: "You are a professional researcher. Your task is to collect key information and data on a given topic and provide a structured research report. The report should contain 5 key points, with each point not exceeding 2 sentences.",
		MemoryConfig: memory.Config{
			Type:     "inmemory",
			Capacity: 5,
		},
		MaxTokens:   1000,
		Temperature: 0.3,
	})
	log.Println("Researcher agent created successfully")

	// 2. Writer Agent - responsible for drafting
	log.Println("Creating copywriter agent...")
	writerAgent := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:         "Copywriter",
		Provider:     provider,
		SystemPrompt: "You are a professional copywriter. Your task is to write engaging content based on the research report. You need to expand on each key point, adding vivid examples and explanations. The generated content should be attractive and easy to read.",
		MemoryConfig: memory.Config{
			Type:     "inmemory",
			Capacity: 5,
		},
		MaxTokens:   1500,
		Temperature: 0.7,
	})
	log.Println("Copywriter agent created successfully")

	// 3. Editor Agent - responsible for review and optimization
	log.Println("Creating editor agent...")
	editorAgent := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:         "Editor",
		Provider:     provider,
		SystemPrompt: "You are a professional editor. Your task is to review and optimize the copy to ensure the content is clear, coherent, and error-free. You should improve wording, structure, and format while maintaining the core information of the original content.",
		MemoryConfig: memory.Config{
			Type:     "inmemory",
			Capacity: 5,
		},
		MaxTokens:   1200,
		Temperature: 0.4,
	})
	log.Println("Editor agent created successfully")

	// Create Writing Agency
	log.Println("Creating Writing Agency...")
	writingAgency := agency.New(agency.Config{
		Name:               "Writing Team",
		SharedInstructions: "Collaborate to complete research, writing, and editing tasks to create high-quality articles",
		Orchestrator:       system.Orchestrator,
	})

	// Add agents to Agency
	log.Println("Adding agents to Agency...")
	writingAgency.AddAgent(researchAgent)
	writingAgency.AddAgent(writerAgent)
	writingAgency.AddAgent(editorAgent)

	// Define communication flow between agents
	log.Println("Defining agent communication flow...")
	err = writingAgency.DefineFlowChart([]agency.Flow{
		{researchAgent},              // Researcher is the entry point
		{researchAgent, writerAgent}, // Researcher can communicate with the copywriter
		{writerAgent, editorAgent},   // Copywriter can communicate with the editor
	})
	if err != nil {
		log.Fatalf("Failed to define flow chart: %v", err)
	}
	log.Println("Agent communication flow defined successfully")

	// Start the orchestrator
	log.Println("Starting orchestrator...")
	err = system.Orchestrator.Start()
	if err != nil {
		log.Fatalf("Failed to start orchestrator: %v", err)
	}
	log.Println("Orchestrator started successfully")

	// Create workflow
	log.Println("========== Creating Writing Workflow ==========")
	topic := "Applications of Artificial Intelligence in Education"
	log.Printf("Research topic: %s", topic)

	writeWorkflow := agency.NewWorkflow("Article Writing Workflow", "Complete an article from research to editing")

	// Add research step
	researchStep := &agency.WorkflowStep{
		ID:          "research",
		Name:        "Research Topic",
		Description: "Collect key information about the topic",
		AgentID:     "Researcher",
		Transform: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
			topic := inputs["input"].(string)
			return fmt.Sprintf("Please research the following topic and provide key information: %s", topic), nil
		},
	}
	log.Println("Adding research step...")
	err = writeWorkflow.AddStep(researchStep)
	if err != nil {
		log.Fatalf("Failed to add research step: %v", err)
	}

	// Add writing step
	writeStep := &agency.WorkflowStep{
		ID:          "write",
		Name:        "Write Article",
		Description: "Write an article based on research results",
		AgentID:     "Copywriter",
		InputFrom:   []string{"research"},
		Transform: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
			research := inputs["research"].(string)
			return fmt.Sprintf("Write an article based on the following research report:\n\n%s", research), nil
		},
	}
	log.Println("Adding writing step...")
	err = writeWorkflow.AddStep(writeStep)
	if err != nil {
		log.Fatalf("Failed to add writing step: %v", err)
	}

	// Add editing step
	editStep := &agency.WorkflowStep{
		ID:          "edit",
		Name:        "Edit Article",
		Description: "Edit and optimize the draft",
		AgentID:     "Editor",
		InputFrom:   []string{"write"},
		Transform: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
			draft := inputs["write"].(string)
			return fmt.Sprintf("Please edit the following article draft:\n\n%s", draft), nil
		},
	}
	log.Println("Adding editing step...")
	err = writeWorkflow.AddStep(editStep)
	if err != nil {
		log.Fatalf("Failed to add editing step: %v", err)
	}

	// Execute workflow
	log.Println("========== Executing Workflow ==========")
	ctx := context.Background()
	log.Println("Starting workflow execution...")
	startTime := time.Now()

	result, err := writeWorkflow.Execute(ctx, writingAgency, topic)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	endTime := time.Now()
	log.Printf("Workflow execution completed, time taken: %v", endTime.Sub(startTime))

	// Output final result
	log.Println("\n========= Final Article =========")
	log.Println(result)
	log.Println("============================")

	// Stop the orchestrator
	log.Println("Stopping orchestrator...")
	err = system.Orchestrator.Stop()
	if err != nil {
		log.Fatalf("Failed to stop orchestrator: %v", err)
	}
	log.Println("Orchestrator stopped")

	log.Println("========== Processing Complete ==========")
}

// Function to truncate string, used for truncating long content in log output
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
