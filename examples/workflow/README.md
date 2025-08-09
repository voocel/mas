# Multi-Agent Workflow Example

This example demonstrates the simple and elegant workflow orchestration system in MAS.

## Features

- **Sequential Workflows**: Chain agents together in a pipeline
- **Parallel Execution**: Run multiple agents concurrently
- **Tool Integration**: Seamlessly integrate tools into workflows
- **Context Sharing**: Agents share data through WorkflowContext

## Usage

```bash
export OPENAI_API_KEY="your-api-key"
go run main.go
```

## API Overview

### Basic Workflow

```go
workflow := mas.NewWorkflow().
    AddNode(mas.NewAgentNode("researcher", researcher)).
    AddNode(mas.NewAgentNode("writer", writer)).
    AddEdge("researcher", "writer").
    SetStart("researcher")

result, err := workflow.Execute(ctx, initialData)
```

### Parallel Execution

```go
parallelNode := mas.NewParallelNode("analysis",
    mas.NewAgentNode("tech", techAgent),
    mas.NewAgentNode("market", marketAgent),
)

workflow := mas.NewWorkflow().
    AddNode(parallelNode).
    SetStart("analysis")
```

### Tool Integration

```go
calculator := mas.NewToolNode("calc", tools.Calculator()).
    WithParams(map[string]any{
        "operation": "multiply",
        "a": 123,
        "b": 456,
    })

workflow := mas.NewWorkflow().
    AddNode(calculator).
    SetStart("calc")
```

## Design Principles

- **Simple**: Minimal API surface, easy to understand
- **Elegant**: Fluent builder pattern for workflow construction
- **Extensible**: Easy to add new node types
- **Safe**: Context-aware execution with proper error handling

## Node Types

- **AgentNode**: Wraps an Agent for LLM-powered processing
- **ToolNode**: Wraps a Tool for external capabilities
- **ParallelNode**: Executes multiple nodes concurrently

## Context Management

The `WorkflowContext` provides thread-safe data sharing between nodes:

- `Get(key)` / `Set(key, value)` for data access
- `AddMessage(role, content)` for conversation history
- Automatic message tracking for debugging
