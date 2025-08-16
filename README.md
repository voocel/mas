# MAS - Lightweight Multi-Agent Framework for Go

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![GoDoc](https://pkg.go.dev/badge/github.com/voocel/mas.svg)](https://pkg.go.dev/github.com/voocel/mas)

[中文](README_CN.md) | English

MAS (Multi-Agent System) is a lightweight, elegant multi-agent framework for Go, designed to make it easy to integrate intelligent agent capabilities into your applications.

## Design Philosophy

- **Simple First**: Minimal API surface with zero-config defaults
- **Easy Integration**: Single import, no architectural constraints
- **Convention over Configuration**: Sensible defaults, minimal setup required
- **Progressive Complexity**: Supports simple to complex use cases

## Features

- **Intelligent Agents**: LLM-powered agents with memory and tools
- **Workflow Orchestration**: State graph-based multi-agent coordination
- **Conditional Routing**: Dynamic workflow paths based on context state
- **Human-in-the-Loop**: Interactive approval and input mechanisms
- **Tool System**: Extensible tool framework with sandbox security
- **Memory Management**: Conversation and summary memory implementations
- **LLM Integration**: Built on [litellm](https://github.com/voocel/litellm) for multiple providers
- **Checkpoint & Recovery**: Advanced workflow persistence and recovery system
  - **Automatic Checkpointing**: Saves state at key points
  - **Smart Recovery**: Resumes from interruption point
  - **Multiple Storage Options**: File, memory, database
  - **Compression Support**: Efficient storage
  - **Error Handling**: Graceful failure recovery
- **Lightweight**: Minimal dependencies, easy to embed
- **Fluent API**: Chain-able configuration methods

## Architecture

```
mas/
├── agent.go            # Core Agent implementation with LLM integration
├── workflow.go         # Workflow orchestration and state management  
├── tool.go             # Tool framework and interface
├── memory.go           # Memory systems (conversation, summary)
├── checkpoint.go       # Checkpoint interfaces and utilities
├── types.go            # Core type definitions and interfaces
├── errors.go           # Error types and handling
├── agent/              # Agent implementation details
│   ├── agent.go        # Core Agent Implementation
│   ├── execution.go    # Tool Invocation Logic
│   └── config.go       # Config
├── workflow/           # Workflow execution engine
│   ├── builder.go      # Workflow Builder
│   ├── context.go      # Workflow Context
│   ├── executor.go     # Execution Engine
│   ├── nodes.go        # All node type implementations
│   └── routing.go      # Conditional Routing
├── llm/                # LLM provider abstraction
│   ├── provider.go     # Provider interface and factories
│   ├── litellm.go      # LiteLLM adapter implementation
│   ├── types.go        # LLM-specific types
│   └── converter.go    # Format conversion utilities
├── memory/             # Memory implementations
│   ├── conversation.go # Conversation memory
│   ├── summary.go      # Summary memory with compression
│   └── config.go       # Memory configuration
├── checkpoint/         # Checkpoint system
│   ├── manager.go      # Checkpoint manager
│   ├── store.go        # Storage interface
│   ├── file.go         # File storage backend
│   ├── memory.go       # In-memory storage
│   ├── redis.go        # Redis storage (+build redis)
│   └── sqlite.go       # SQLite storage (+build sqlite)
├── tools/              # Built-in tool ecosystem
├── examples/           # Usage examples
│   ├── basic/          # Basic agent usage
│   ├── workflow/       # Multi-agent workflows
│   ├── tools/          # Custom tools and multiple tools
│   ├── baseurl/        # Custom API endpoints
│   ├── checkpoint/     # Checkpoint and recovery
│   └── verify/         # Installation verification
└── internal/           # Internal utilities
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
    "os"
    
    "github.com/voocel/mas"
)

func main() {
    // Create an agent with minimal setup
    agent := mas.NewAgent("o3", os.Getenv("OPENAI_API_KEY"))
    
    // Chat with the agent
    response, err := agent.Chat(context.Background(), "Hello! How are you?")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(response)
}
```

### With Tools and Memory

```go
func main() {
    // Create a custom tool
    greetingTool := mas.NewSimpleTool("greeting", "Generate a greeting", 
        func(ctx context.Context, params map[string]any) (any, error) {
            return "Hello, World!", nil
        })

    agent := mas.NewAgent("gpt-4", os.Getenv("OPENAI_API_KEY")).
        WithTools(greetingTool).
        WithMemory(mas.NewConversationMemory(10)).
        WithSystemPrompt("You are a helpful research assistant.")
    
    response, _ := agent.Chat(context.Background(), "Use the greeting tool")
    fmt.Println(response)
}
```

### Custom Base URL (DeepSeek, Ollama, Azure OpenAI, etc.)

```go
func main() {
    // Using custom OpenAI-compatible API
    config := mas.AgentConfig{
        Name:        "DeepSeekAgent",
        Model:       "deepseek-chat",
        APIKey:      os.Getenv("DEEPSEEK_API_KEY"),
        BaseURL:     "https://api.deepseek.com/v1",  // Custom endpoint
        Temperature: 0.7,
        MaxTokens:   1000,
    }
    
    agent := mas.NewAgentWithConfig(config)
    response, _ := agent.Chat(context.Background(), "Hello!")
    fmt.Println(response)
}
```

**Supported Custom Endpoints:**
- DeepSeek: `https://api.deepseek.com/v1`
- Local Ollama: `http://localhost:11434/v1`
- Azure OpenAI: `https://your-resource.openai.azure.com/openai/deployments/gpt-4`
- Any OpenAI-compatible API

## Examples

Run the examples to see MAS in action:

```bash
# Basic agent usage
cd examples/basic && go run main.go

# Multi-agent workflows  
cd examples/workflow && go run main.go

# Custom tools and multiple tools
cd examples/tools && go run main.go

# Custom base URL configuration
cd examples/baseurl && go run main.go

# Verify framework installation
cd examples/verify && go run main.go
```

### Multi-Agent Workflows

```go
func main() {
    // Create specialized agents
    researcher := mas.NewAgent("gpt-4", apiKey).
        WithSystemPrompt("You are a researcher.")

    writer := mas.NewAgent("gpt-4", apiKey).
        WithSystemPrompt("You are a writer.")

    // Create workflow with state graph
    workflow := mas.NewWorkflow().
        AddNode(mas.NewAgentNode("researcher", researcher)).
        AddNode(mas.NewAgentNode("writer", writer)).
        AddEdge("researcher", "writer").
        SetStart("researcher")

    result, err := workflow.Execute(context.Background(), map[string]any{
        "input": "Research AI trends and write a summary",
    })
}
```
// Simple conditional routing
workflow.AddConditionalRoute("classifier",
    func(ctx *mas.WorkflowContext) bool {
        output := ctx.Get("output")
        return strings.Contains(fmt.Sprintf("%v", output), "technical")
    },
    "tech_expert", "biz_expert")

// Multi-branch conditions
workflow.AddConditionalEdge("router",
    mas.When(func(ctx *mas.WorkflowContext) bool {
        return ctx.Get("score").(int) > 8
    }, "approve"),
    mas.When(func(ctx *mas.WorkflowContext) bool {
        return ctx.Get("score").(int) > 5
    }, "review"),
    mas.When(func(ctx *mas.WorkflowContext) bool { return true }, "reject"),
)
```

### Human-in-the-Loop

```go
// Console input with timeout and validation
humanProvider := mas.NewConsoleInputProvider()
humanNode := mas.NewHumanNode("reviewer", "Please review the content:", humanProvider).
    WithOptions(
        mas.WithTimeout(5*time.Minute),
        mas.WithValidator(func(input string) error {
            if len(input) < 10 {
                return errors.New("feedback too short")
            }
            return nil
        }),
    )

// Custom input provider (Web, API, etc.)
type WebInputProvider struct{}
func (p *WebInputProvider) RequestInput(ctx context.Context, prompt string, options ...mas.HumanInputOption) (*mas.HumanInput, error) {
    // Implement web-based input collection
}
```

## Examples

The [`examples/`](examples/) directory contains comprehensive examples:

- **[Basic Usage](examples/basic/)** - Simple agent interactions and configuration
- **[Tools Usage](examples/tools/)** - Built-in and custom tools with sandbox
- **[Workflow Orchestration](examples/workflow/)** - Multi-agent workflows and coordination

Run examples:

```bash
cd examples/basic
export OPENAI_API_KEY="your-api-key"
go run main.go
```

## Built-in Tools

MAS comes with useful tools out of the box:

### Math & Computation
- **Calculator** - Basic arithmetic operations
- **Advanced Calculator** - Expression evaluation

### Web & Network
- **Web Search** - Search engines integration
- **HTTP Request** - REST API calls
- **Web Scraper** - Extract web content
- **Domain Info** - WHOIS, DNS, SSL information

### File Operations
- **File Reader/Writer** - Read and write files with sandbox support
- **Directory Lister** - Browse filesystem with path restrictions
- **File Info** - Get file metadata with access control

### Data Processing
- **JSON Parser** - Parse and manipulate JSON
- **URL Shortener** - Create short URLs

## Memory Systems

### Conversation Memory
```go
// Remember last 10 messages
memory := memory.Conversation(10)

// With custom configuration
memory := memory.ConversationWithConfig(mas.MemoryConfig{
    MaxMessages: 50,
    TTL: 24 * time.Hour,
})
```

### Persistent Memory
```go
// Saves to disk automatically
memory := memory.Persistent(100, "./chat_history.json")
```

### Advanced Memory
```go
// Thread-safe shared memory
shared := memory.ThreadSafe(memory.Conversation(20))

// Multi-tier with fast/slow storage
multiTier := memory.MultiTier(
    memory.Conversation(10),    // Fast memory
    memory.Persistent(1000, "./history.json"), // Slow memory
    10, // Fast memory limit
)
```



## Custom Tools

Create your own tools easily:

```go
func MyCustomTool() mas.Tool {
    schema := &mas.ToolSchema{
        Type: "object",
        Properties: map[string]*mas.PropertySchema{
            "input": mas.StringProperty("Input text to process"),
        },
        Required: []string{"input"},
    }
    
    return mas.NewTool(
        "my_tool",
        "Description of what this tool does",
        schema,
        func(ctx context.Context, params map[string]any) (any, error) {
            input := params["input"].(string)
            // Your tool logic here
            return map[string]any{
                "result": "processed: " + input,
            }, nil
        },
    )
}
```

## Architecture

```
mas/
├── agent.go           # Core agent implementation
├── tool.go            # Tool interface and registry
├── memory.go          # Memory interfaces
├── workflow.go        # Multi-agent workflow orchestration
├── llm/               # LLM provider integration
├── tools/             # Built-in tools
├── memory/            # Memory implementations
└── examples/          # Usage examples
```

## LLM Support

Built on [litellm](https://github.com/voocel/litellm) for broad LLM provider support:

- **OpenAI** - o3, o4-mini, GPT-4.1, GPT-4o
- **Anthropic** - Claude 4 Opus, Claude 4 Sonnet, Claude 3.7 Sonnet
- **Google** - Gemini 2.5 Pro, Gemini 2.5 Flash
- **And more...**

```go
// Automatic provider detection
agent := mas.NewAgent("o3", apiKey)
agent := mas.NewAgent("claude-4-sonnet", apiKey)
agent := mas.NewAgent("gemini-2.5-pro", apiKey)
```

## Configuration

### Agent Configuration
```go
config := mas.AgentConfig{
    Name:         "MyAgent",
    Model:        "o3",
    APIKey:       apiKey,
    SystemPrompt: "You are a helpful assistant",
    Temperature:  0.7,
    MaxTokens:    2000,
    Tools:        []mas.Tool{tools.Calculator()},
    Memory:       memory.Conversation(10),
}

agent := mas.NewAgentWithConfig(config)
```

### Fluent Configuration
```go
agent := mas.NewAgent("o3", apiKey).
    WithSystemPrompt("You are an expert...").
    WithTemperature(0.3).
    WithMaxTokens(1000).
    WithTools(tools.Calculator()).
    WithMemory(memory.Conversation(5))
```

## Testing

```bash
go test ./...
```

## Documentation

- [Design Document](CLAUDE.md) - Architecture and design decisions
- [API Reference](https://pkg.go.dev/github.com/voocel/mas) - Complete API documentation
- [Examples](examples/) - Practical usage examples

## Roadmap

### High Priority
- [x] **Conditional Routing** - Dynamic workflow paths based on context state
- [x] **Human-in-the-Loop** - Interactive approval and input mechanisms
- [ ] **Loop Detection & Control** - Smart loop handling and cycle prevention

### Medium Priority
- [ ] **Checkpoint & Recovery** - Workflow state persistence and resumption
- [ ] **Role-based Agents** - Built-in role system with predefined behaviors
- [ ] **Advanced Tool Integration** - Tool chaining and conditional tool usage

### Low Priority
- [ ] **Monitoring & Observability** - Built-in tracing and metrics
- [ ] **Visual Workflow Designer** - Web-based workflow builder
- [ ] **Cloud Integration** - Native support for major cloud platforms

## Contributing

Contributions are welcome! Please feel free to:

- Report bugs
- Suggest features
- Submit pull requests
- Improve documentation

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Related Projects

- [litellm](https://github.com/voocel/litellm) - Unified LLM client library
- [OpenAI Go](https://github.com/sashabaranov/go-openai) - OpenAI API client
