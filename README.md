# MAS - Flexible Multi-Agent Framework | [中文](README.zh-CN.md)

MAS (Multi-Agent System) is a flexible, modular multi-agent framework written in Go, designed specifically for building intelligent applications based on large language models.

## Framework Features

- **Open Architecture**: Core components defined through interfaces, supporting customization and replacement
- **LLM Integration**: Built-in support for OpenAI API, easily extensible to support other models
- **Agent System**: Provides basic agent interfaces and implementations, supporting perception-thinking-action loops
- **Tool Integration**: Flexible tool definition and invocation mechanisms allowing agents to call external functions
- **Memory System**: Supports various memory types, including short-term and long-term memory
- **Knowledge Graph**: Structured knowledge representation of entities and relationships, enhancing agents' long-term memory and reasoning capabilities
- **Communication Mechanism**: Message passing system between agents, supporting point-to-point and broadcast communication
- **Task Orchestration**: Unified management of task definition, allocation, and execution

## Project Structure

```
mas/
├── agent/           # Agent definitions and implementations
├── communication/   # Communication system
├── knowledge/       # Knowledge graph system
├── llm/             # Large language model integration
├── memory/          # Memory system
├── orchestrator/    # Task orchestration
├── tools/           # Tool system
├── examples/        # Example projects
└── go.mod           # Go module definition
```

## Quick Start

### Installation

```bash
go get github.com/voocel/mas
```

### Basic Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/tools"
)

func main() {
	// Initialize LLM provider
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
		30*time.Second,
	)

	// Create agent
	assistant := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:        "assistant",
		LLMProvider: provider,
		SystemPrompt: `You are a helpful assistant.`,
	})

	// Create tools
	toolRegistry := tools.NewRegistry()
	// Register tools...

	// Run agent
	result, err := assistant.Process(context.Background(), "Hello, please introduce yourself")
	if err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	fmt.Println(result)
}
```

## Simple Example Projects

### 1. Chat Assistant

A basic multi-turn conversation chat assistant that demonstrates how to maintain conversation history and state.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/llm"
)

func main() {
	// Initialize LLM provider
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4o",
		30*time.Second,
	)

	// Create conversation memory
	conversationMemory := memory.NewConversationMemory(10) // Save the last 10 rounds of conversation
	
	// Create chat agent
	chatbot := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:         "Chatbot",
		Provider:     provider,
		SystemPrompt: "You are a friendly chat assistant who answers questions in a concise and clear manner.",
		Memory:       conversationMemory,
		MaxTokens:    1000,
		Temperature:  0.7,
	})

	// Main conversation loop
	fmt.Println("Chat assistant started, type 'exit' to quit")
	for {
		fmt.Print("> ")
		var input string
		fmt.Scanln(&input)
		
		if input == "exit" {
			break
		}
		
		response, err := chatbot.Process(context.Background(), input)
		if err != nil {
			log.Printf("Processing failed: %v", err)
			continue
		}
		
		fmt.Println(response)
	}
}
```

Run the example:

```bash
cd examples/chat_assistant
export OPENAI_API_KEY="your_api_key"
go run main.go
```

### 2. Task Planner

Demonstrates how to create a simple task planning agent system that breaks down complex goals into executable steps.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
)

func main() {
	// Initialize LLM provider
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
		30*time.Second,
	)

	// Create planner agent
	planner := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Planner",
		Provider: provider,
		SystemPrompt: `You are a task planning expert. Your job is to break down the user's goal into detailed step-by-step plans.
Each step should be specific, actionable, and in logical order.`,
		MaxTokens:   2000,
		Temperature: 0.2,
	})

	// Create executor agent
	executor := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Executor",
		Provider: provider,
		SystemPrompt: `You are an execution expert. Your job is to provide detailed execution suggestions for each planned step.
For each step, provide specific action guidelines, possible resources, and considerations.`,
		MaxTokens:   1500,
		Temperature: 0.3,
	})

	// User goal
	goal := "Learn Go language and develop a web application with it"

	// First use the planning agent to generate a plan
	fmt.Println("Generating plan for goal:", goal)
	planResult, err := planner.Process(context.Background(), goal)
	if err != nil {
		log.Fatalf("Planning failed: %v", err)
	}

	plan, ok := planResult.(string)
	if !ok {
		log.Fatalf("Planning result type error")
	}

	fmt.Println("\n===== PLAN =====")
	fmt.Println(plan)

	// Then use the execution agent to provide execution suggestions
	executionRequest := fmt.Sprintf("Provide detailed execution suggestions for the following plan:\n\n%s", plan)
	executionResult, err := executor.Process(context.Background(), executionRequest)
	if err != nil {
		log.Fatalf("Execution suggestion generation failed: %v", err)
	}

	fmt.Println("\n===== EXECUTION SUGGESTIONS =====")
	fmt.Println(executionResult)
}
```

Run the example:

```bash
cd examples/task_planner
export OPENAI_API_KEY="your_api_key"
go run main.go
```

### 3. Collaborative Writing

Demonstrates how to create a collaborative writing system where multiple agents work together to complete an article.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/communication"
	"github.com/voocel/mas/llm"
)

func main() {
	// Initialize LLM provider
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
		30*time.Second,
	)

	// Create communication bus
	bus := communication.NewInMemoryBus()

	// Create editor agent
	editor := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Editor",
		Provider: provider,
		SystemPrompt: `You are a senior editor. Your task is to review and refine content provided by writers.
Focus on logical coherence, structure, style, and overall quality. Provide constructive suggestions for improvement.`,
		MaxTokens:   1500,
		Temperature: 0.3,
	})

	// Create writer agent
	writer := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Writer",
		Provider: provider,
		SystemPrompt: `You are a creative writer. Your task is to create original content based on the given topic.
Focus on creativity, interest, and expressiveness in your content.`,
		MaxTokens:   2000,
		Temperature: 0.7,
	})

	// Create polisher agent
	polisher := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Polisher",
		Provider: provider,
		SystemPrompt: `You are a text polishing expert. Your task is to make final refinements to the draft.
Enhance the beauty of language, correct any grammar or punctuation errors, and ensure consistent overall style.`,
		MaxTokens:   1500,
		Temperature: 0.4,
	})

	// Start collaborative writing process
	topic := "Applications of Artificial Intelligence in Daily Life"
	ctx := context.Background()

	fmt.Println("Starting collaborative writing, topic:", topic)
	
	// Step 1: Writer creates initial draft
	fmt.Println("\n1. Writer is creating initial draft...")
	initialDraft, err := writer.Process(ctx, fmt.Sprintf("Please write an article of about 800 words on the topic '%s'", topic))
	if err != nil {
		log.Fatalf("Creating initial draft failed: %v", err)
	}
	
	fmt.Println("\n===== INITIAL DRAFT =====")
	fmt.Println(initialDraft)
	
	// Step 2: Editor reviews and provides modification suggestions
	fmt.Println("\n2. Editor is reviewing...")
	editorFeedback, err := editor.Process(ctx, fmt.Sprintf("Please review and provide suggestions for improvement:\n\n%s", initialDraft))
	if err != nil {
		log.Fatalf("Editor review failed: %v", err)
	}
	
	fmt.Println("\n===== EDITOR SUGGESTIONS =====")
	fmt.Println(editorFeedback)
	
	// Step 3: Writer revises based on suggestions
	fmt.Println("\n3. Writer is revising...")
	revisedDraft, err := writer.Process(ctx, fmt.Sprintf("Please revise the article based on the editor's suggestions:\n\nOriginal:\n%s\n\nEditor's suggestions:\n%s", initialDraft, editorFeedback))
	if err != nil {
		log.Fatalf("Revision failed: %v", err)
	}
	
	fmt.Println("\n===== REVISED DRAFT =====")
	fmt.Println(revisedDraft)
	
	// Step 4: Final polishing
	fmt.Println("\n4. Polishing expert is making final refinements...")
	finalDraft, err := polisher.Process(ctx, fmt.Sprintf("Please make final refinements to this article:\n\n%s", revisedDraft))
	if err != nil {
		log.Fatalf("Polishing failed: %v", err)
	}
	
	fmt.Println("\n===== FINAL DRAFT =====")
	fmt.Println(finalDraft)
	
	fmt.Println("\nCollaborative writing completed!")
}
```

Run the example:

```bash
cd examples/collaborative_writing
export OPENAI_API_KEY="your_api_key"
go run main.go
```

### Knowledge Graph Agent

The example project in `examples/knowledge_agent/` demonstrates how to integrate knowledge graphs into agents, enhancing their knowledge representation and reasoning capabilities.

This example demonstrates:
- How to create and use knowledge graphs
- How to define entities and relationships
- How to utilize knowledge graphs in agents to provide information
- How to query relevant knowledge to enhance answer quality

Run the example:

```bash
cd examples/knowledge_agent
export OPENAI_API_KEY="your_api_key"
go run main.go
```

## Extending the Framework

The MAS framework is designed to be highly extensible. You can:

1. **Implement custom agents**: Inherit from the base agent interface, customize behavior according to needs
2. **Add new tools**: Implement the tool interface to provide new capabilities to agents
3. **Extend LLM support**: Add new language model providers
4. **Customize memory models**: Implement the memory interface to provide different types of memory systems
5. **Enhance knowledge graphs**: Implement more complex graph database backends or optimize query performance
6. **Enhance communication mechanisms**: Support more complex inter-agent communication patterns

## Contribution

Contributions of code, issue reports, or feature suggestions are welcome.

## License

MIT 