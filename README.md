# MAS

MAS is a modern, high-performance Go multi-agent system framework designed for building intelligent applications with native Go performance and type safety.

![Architecture](./docs/architecture.md)

[Documentation](./docs/) | [Examples](./examples/) | [Chinese Version](./README_CN.md)

## Installation

```bash
go get github.com/voocel/mas
```

## Quick Start

**Set up environment variables:**

```bash
export LLM_API_KEY="your-api-key"
export LLM_MODEL="gpt-5"
export LLM_BASE_URL="https://api.openai.com/v1"  # optional
```

**Basic usage:**

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/voocel/mas/agent"
    "github.com/voocel/mas/llm"
    "github.com/voocel/mas/runtime"
    "github.com/voocel/mas/schema"
)

func main() {
    // Create LLM model
    model, _ := llm.NewLiteLLMAdapter(
        os.Getenv("LLM_MODEL"),
        os.Getenv("LLM_API_KEY"),
        os.Getenv("LLM_BASE_URL"),
    )

    // Create agent
    agent := agent.NewAgent("assistant", "AI Assistant", model)

    // Execute
    ctx := runtime.NewContext(context.Background(), "session-1", "trace-1")
    response, _ := agent.Execute(ctx, schema.Message{
        Role:    schema.RoleUser,
        Content: "Hello, how are you?",
    })

    fmt.Printf("Agent: %s\n", response.Content)
}
```

**Agent with tools:**

```go
import "github.com/voocel/mas/tools/builtin"

// Create agent with calculator tool
calculator := builtin.NewCalculator()
agent := agent.NewAgent("math-agent", "Math Assistant", model,
    agent.WithTools(calculator),
)

response, _ := agent.Execute(ctx, schema.Message{
    Role:    schema.RoleUser,
    Content: "Calculate 15 * 8 + 7",
})
```

**Multi-agent collaboration:**

```go
import "github.com/voocel/mas/orchestrator"

// Create orchestrator
orch := orchestrator.NewOrchestrator()

// Add agents
orch.AddAgent("researcher", researcher)
orch.AddAgent("writer", writer)

// Execute with routing
result, _ := orch.Execute(ctx, orchestrator.ExecuteRequest{
    Input: schema.Message{
        Role:    schema.RoleUser,
        Content: "Research and write about AI trends",
    },
    Type: orchestrator.ExecuteTypeAuto, // Auto-route to best agent
})
```

**Workflow orchestration:**

```go
import "github.com/voocel/mas/workflows"

// Create workflow steps
preprocessStep := workflows.NewFunctionStep(
    workflows.NewStepConfig("preprocess", "Data preprocessing"),
    func(ctx runtime.Context, input schema.Message) (schema.Message, error) {
        // Processing logic
        return processedMessage, nil
    },
)

// Build chain workflow
workflow := workflows.NewChainBuilder("data-pipeline", "Data processing pipeline").
    Then(preprocessStep).
    Then(analysisStep).
    Then(summaryStep).
    Build()

// Execute workflow
orch.AddWorkflow("data-pipeline", workflow)
result, _ := orch.Execute(ctx, orchestrator.ExecuteRequest{
    Input:  inputMessage,
    Target: "data-pipeline",
    Type:   orchestrator.ExecuteTypeWorkflow,
})
```

## Core Concepts

### Agent vs Workflow

**Agent** - Autonomous intelligent entities with:
- **Smart Decision Making**: LLM-powered reasoning and tool selection
- **Dynamic Behavior**: Adapts to different scenarios and contexts
- **Tool Integration**: Access to various tools and capabilities
- **Memory Management**: Maintains conversation history and context

**Workflow** - Structured process orchestration with:
- **Deterministic Execution**: Predefined steps and control flow
- **Parallel Processing**: Concurrent execution of independent tasks
- **Conditional Branching**: Dynamic routing based on conditions
- **Composability**: Steps can be functions, agents, or other workflows

**Flexible Composition**:
- **Agent in Workflow**: Workflow steps can be agents
- **Workflow in Agent**: Agents can call workflows as tools
- **Hybrid Execution**: Mix agents and workflows in the same task

## Examples & Integrations


## License

Apache License 2.0 License
