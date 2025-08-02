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
- **Tool System**: Extensible tool framework for external capabilities
- **Memory Management**: Conversation and summary memory implementations
- **Team Collaboration**: Multi-agent workflows and coordination
- **LLM Integration**: Built on [litellm](https://github.com/voocel/litellm) for multiple providers
- **Lightweight**: Minimal dependencies, easy to embed
- **Fluent API**: Chain-able configuration methods

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
import (
    "github.com/voocel/mas"
    "github.com/voocel/mas/tools"
    "github.com/voocel/mas/memory"
)

func main() {
    agent := mas.NewAgent("o3", os.Getenv("OPENAI_API_KEY")).
        WithTools(tools.Calculator(), tools.WebSearch()).
        WithMemory(memory.Conversation(10)).
        WithSystemPrompt("You are a helpful research assistant.")
    
    response, err := agent.Chat(context.Background(), 
        "Calculate 15% of 250, then search for information about percentages")
    // Agent will use calculator tool and web search automatically
}
```

### Team Collaboration

```go
func main() {
    // Create specialized agents
    researcher := mas.NewAgent("gemini-2.5-pro", apiKey).
        WithSystemPrompt("You are a researcher. Gather key information.").
        WithTools(tools.WebSearch())
    
    writer := mas.NewAgent("claude-4-sonnet", apiKey).
        WithSystemPrompt("You are a writer. Create engaging content.")
    
    // Create a team workflow
    team := mas.NewTeam().
        Add("researcher", researcher).
        Add("writer", writer).
        WithFlow("researcher", "writer")
    
    result, err := team.Execute(context.Background(), 
        "Research and write about renewable energy benefits")
}
```

## Examples

The [`examples/`](examples/) directory contains comprehensive examples:

- **[Basic Usage](examples/basic/)** - Simple agent interactions and configuration
- **[Tools Usage](examples/tools/)** - Built-in and custom tools
- **[Team Collaboration](examples/team/)** - Multi-agent workflows

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
- **File Reader/Writer** - Read and write files
- **Directory Lister** - Browse filesystem
- **File Info** - Get file metadata

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

## Team Patterns

### Sequential Processing
```go
team := mas.NewTeam().
    Add("analyzer", analyzerAgent).
    Add("writer", writerAgent).
    Add("editor", editorAgent).
    WithFlow("analyzer", "writer", "editor")
```

### Parallel Processing
```go
team := mas.NewTeam().
    Add("tech", techAgent).
    Add("business", businessAgent).
    Add("risk", riskAgent).
    WithFlow("tech", "business", "risk").
    WithParallel(true)
```

### Shared Memory
```go
sharedMemory := memory.ThreadSafe(memory.Conversation(30))

team := mas.NewTeam().
    Add("agent1", agent1).
    Add("agent2", agent2).
    SetSharedMemory(sharedMemory)
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
├── team.go            # Multi-agent collaboration
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
