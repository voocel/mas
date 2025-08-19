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
- **Hierarchical Cognitive Architecture**: Brain-Cerebellum inspired layered intelligence
  - **Four-Layer Processing**: Reflex → Cerebellum → Cortex → Meta cognitive layers
  - **Automatic Layer Selection**: Intelligent selection of optimal processing layer
  - **Skill Library System**: Pluggable cognitive capabilities and learned behaviors
  - **Cognitive State Monitoring**: Real-time cognitive state tracking and introspection
  - **Adaptive Processing**: Dynamic switching between reactive and deliberative modes
- **Autonomous Goal Management**: Self-directed task execution with intelligent strategies
  - **Goal Decomposition**: Automatic breakdown of complex goals into actionable steps
  - **Multiple Strategies**: Sequential, parallel, priority-based, and adaptive execution
  - **Progress Monitoring**: Real-time tracking of goal achievement and strategy adjustment
  - **Learning Integration**: Performance insights and continuous improvement
- **Learning & Adaptation**: Continuous improvement through experience and self-reflection
  - **Experience Recording**: Detailed logging of all interactions and outcomes
  - **Pattern Recognition**: Identification of successful and failed behavior patterns
  - **Self-Reflection**: Agent's ability to analyze and improve its own behavior
  - **Performance Prediction**: Predicting success probability of actions based on history
  - **Strategy Optimization**: Dynamic adjustment of decision strategies based on learning
- **Dynamic Collaboration Topology**: Intelligent multi-agent network organization
  - **Seven Topology Types**: Star, Chain, Mesh, Hierarchy, Hub, Ring, and Adaptive patterns
  - **Six Collaboration Modes**: Competitive, Cooperative, Delegation, Consensus, Specialization, and Swarm
  - **Intelligent Task Distribution**: Capability and load-based optimal task assignment
  - **Real-time Performance Analysis**: Comprehensive network metrics and bottleneck prediction
  - **Automatic Optimization**: Dynamic topology restructuring based on performance criteria
  - **Load Balancing**: Smart redistribution to prevent bottlenecks and optimize efficiency
- **Event System**: Real-time observability and monitoring
  - **Real-time Events**: Live execution tracking and progress updates
  - **Performance Monitoring**: Built-in metrics and performance analysis
  - **Error Tracking**: Detailed error context and debugging information
  - **Enterprise Integration**: Easy integration with monitoring systems
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
├── event.go            # Event system for real-time observability
├── cognitive.go        # Hierarchical cognitive architecture (Brain-Cerebellum)
├── autonomous.go       # Autonomous goal management and execution strategies
├── learning.go         # Learning and adaptation mechanisms
├── topology.go         # Dynamic collaboration topology management
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
├── skills/             # Cognitive skill implementations
│   └── basic.go        # Math, text analysis, planning skills  
├── examples/           # Usage examples
│   ├── basic/          # Basic agent usage
│   ├── workflow/       # Multi-agent workflows
│   ├── tools/          # Custom tools and multiple tools
│   ├── cognitive/      # Hierarchical cognitive architecture
│   ├── autonomous/     # Autonomous goal management
│   ├── learning/       # Learning and adaptation
│   ├── topology/       # Dynamic collaboration topology
│   ├── events/         # Event system and real-time monitoring
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

### With Advanced Agent Capabilities (Optional)

```go
import "github.com/voocel/mas/skills"

func main() {
    // Create advanced agent with full AI capabilities
    agent := mas.NewAgent("gpt-4.1", os.Getenv("OPENAI_API_KEY")).
        WithSystemPrompt("You are an intelligent assistant.").
        WithSkills(
            skills.MathSkill(),         // Cerebellum layer: automatic math
            skills.TextAnalysisSkill(), // Cortex layer: complex analysis
            skills.QuickResponseSkill(), // Reflex layer: immediate responses
            skills.PlanningSkill(),     // Meta layer: high-level planning
        ).
        SetCognitiveMode(mas.AutomaticMode).     // Auto-select optimal layer
        WithGoalManager(mas.NewGoalManager()).   // Enable autonomous goals
        WithLearningEngine(mas.NewLearningEngine()) // Enable learning
    
    // Autonomous goal execution
    goal := mas.NewGoal("research_project", "Research AI trends and create report", mas.HighPriority)
    agent.AddGoal(context.Background(), goal)
    agent.StartAutonomous(context.Background()) // Runs autonomously
    
    // Cognitive skill execution (Cerebellum layer)
    result, _ := agent.ExecuteSkill(context.Background(), "math_calculation", 
        map[string]interface{}{"expression": "25 * 4 + 10"})
    fmt.Printf("Math result: %v\n", result)
    
    // Learning from experience
    agent.RecordExperience(context.Background(), mas.NewExperience(
        mas.TaskExecution, "completed math task", true, 0.95, nil))
    
    // Self-reflection for improvement
    reflection, _ := agent.SelfReflect(context.Background(), 
        "How can I improve my math calculation performance?")
    fmt.Printf("Self-reflection: %s\n", reflection.Insights)
    
    // Monitor learning metrics
    metrics := agent.GetLearningMetrics()
    fmt.Printf("Learning Rate: %.2f, Adaptation Rate: %.2f\n", 
        metrics.LearningRate, metrics.AdaptationRate)
}
```

### With Dynamic Collaboration Topology (Optional)

```go
func main() {
    // Create dynamic topology for multi-agent collaboration
    topology := mas.NewDynamicTopology(mas.AdaptiveTopology, mas.SwarmMode)
    
    // Create specialized agents
    coordinator := mas.NewAgent("gpt-4", apiKey).
        WithSystemPrompt("You are a project coordinator.")
    specialist := mas.NewAgent("gpt-4", apiKey).
        WithSystemPrompt("You are a domain specialist.")
    worker := mas.NewAgent("gpt-4", apiKey).
        WithSystemPrompt("You are a task executor.")
    
    // Add agents as topology nodes with roles and capabilities
    topology.AddNode(mas.NewTopologyNode(coordinator, mas.CoordinatorRole, 
        []string{"planning", "coordination"}))
    topology.AddNode(mas.NewTopologyNode(specialist, mas.SpecialistRole, 
        []string{"analysis", "research"}))
    topology.AddNode(mas.NewTopologyNode(worker, mas.WorkerRole, 
        []string{"execution", "processing"}))
    
    // Intelligent task distribution
    task := mas.NewCollaborationTask("data_analysis", 3, 
        []string{"analysis", "coordination"})
    assignment, _ := topology.DistributeTask(context.Background(), task)
    fmt.Printf("Task assigned to: %v (Coordinator: %s)\n", 
        assignment.AssignedTo, assignment.Coordinator)
    
    // Adaptive topology optimization
    workload := &mas.WorkloadPattern{
        TaskTypes:        []string{"analysis", "processing"},
        IntensityProfile: map[string]float64{"analysis": 0.9},
        TimePattern:      "peak",
    }
    topology.AdaptToWorkload(context.Background(), workload)
    
    // Performance analysis
    analysis, _ := topology.AnalyzePerformance(context.Background())
    fmt.Printf("Network Efficiency: %.2f, Recommended Topology: %s\n", 
        analysis.EfficiencyScore, analysis.OptimalTopology)
}
```

### With Event System (Optional)

```go
func main() {
    // Create event bus for observability
    eventBus := mas.NewEventBus()
    
    agent := mas.NewAgent("gpt-4.1", os.Getenv("OPENAI_API_KEY")).
        WithSystemPrompt("You are a helpful assistant.").
        WithEventBus(eventBus)  // Enable real-time events
    
    // Subscribe to events for monitoring
    eventBus.Subscribe(mas.EventAgentChatStart, func(ctx context.Context, event mas.Event) error {
        fmt.Printf("Chat started: %s\n", event.Data["message"])
        return nil
    })
    
    eventBus.Subscribe(mas.EventToolStart, func(ctx context.Context, event mas.Event) error {
        fmt.Printf("Tool executing: %s\n", event.Data["tool_name"])
        return nil
    })
    
    // Same API, enhanced with real-time observability
    response, _ := agent.Chat(context.Background(), "Hello!")
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

# Event system and real-time monitoring
cd examples/events && go run main.go

# Hierarchical cognitive architecture
cd examples/cognitive && go run main.go

# Autonomous goal management
cd examples/autonomous && go run main.go

# Learning and adaptation
cd examples/learning && go run main.go

# Dynamic collaboration topology
cd examples/topology && go run main.go

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
- **[Cognitive Architecture](examples/cognitive/)** - Hierarchical cognitive layers and skills
- **[Autonomous Agents](examples/autonomous/)** - Goal-driven autonomous behavior
- **[Learning Systems](examples/learning/)** - Self-improving agents with experience learning
- **[Dynamic Topology](examples/topology/)** - Intelligent multi-agent collaboration networks

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

├── context/                        # 上下文工程核心模块 🎯
│   ├── engine/                     # 上下文引擎
│   │   ├── engine.go              # ContextEngine接口和实现
│   │   ├── state.go               # ContextState管理
│   │   └── checkpoint.go          # 检查点管理
│   │
│   ├── strategy/                   # 四大策略实现
│   │   ├── strategy.go            # 策略接口定义
│   │   ├── write.go               # Write策略：Scratchpad、Memory
│   │   ├── select.go              # Select策略：相关信息选择
│   │   ├── compress.go            # Compress策略：压缩和摘要
│   │   ├── isolate.go             # Isolate策略：上下文隔离
│   │   └── adaptive.go            # 自适应策略组合
│   │
│   ├── memory/                     # 记忆管理
│   │   ├── memory.go              # Memory接口（保持现有）
│   │   ├── episodic.go            # 情景记忆
│   │   ├── semantic.go            # 语义记忆
│   │   ├── procedural.go          # 程序记忆
│   │   └── vector_store.go        # 向量存储
│   │
│   ├── shared/                     # 共享上下文
│   │   ├── shared_context.go      # SharedContext接口
│   │   ├── coordinator.go         # 多Agent协调器
│   │   └── communication.go       # Agent间通信
│   │
│   └── types.go                   # 上下文相关类型定义
│
├── workflow/                       # 工作流模块
│   ├── workflow.go                # 现有工作流（保持兼容）
│   ├── multi_agent.go             # 多Agent工作流
│   ├── command.go                 # AgentCommand实现
│   ├── handoff.go                 # Handoff机制
│   ├── human_node.go              # Human-in-the-Loop节点
│   └── types.go                   # 工作流类型定义