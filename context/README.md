# Context Engineering Module

The Context Engineering module is the core of MAS framework's intelligent context management system. It implements four fundamental strategies for managing context in multi-agent systems: **Write**, **Select**, **Compress**, and **Isolate**.

## 🎯 Overview

Context engineering is crucial for multi-agent systems because:
- **Memory Limitations**: LLMs have finite context windows
- **Information Overload**: Too much context can degrade performance
- **Agent Coordination**: Multiple agents need shared and isolated contexts
- **Persistence**: Important information must survive context window limits

## 🏗️ Architecture

```
context/
├── engine/          # Core context engine
├── strategy/        # Four context strategies + adaptive
├── memory/          # Memory management
├── shared/          # Multi-agent coordination & communication
└── types.go         # Type definitions
```

## 🧠 Core Strategies

### 1. Write Strategy
**Purpose**: Persist information outside the context window

**Features**:
- **Scratchpad Management**: Store working memory
- **Memory Persistence**: Save important information to long-term memory
- **Auto-Summarization**: Generate conversation summaries
- **Importance Scoring**: Automatically assess information importance

**Usage**:
```go
writeStrategy := strategy.NewWriteStrategy(memoryStore)
contextEngine.RegisterStrategy(writeStrategy)
```

### 2. Select Strategy
**Purpose**: Choose relevant information from memory and knowledge

**Features**:
- **Memory Retrieval**: Find relevant past experiences
- **Tool Selection**: Choose appropriate tools for tasks
- **Knowledge Filtering**: Select relevant knowledge items
- **Semantic Search**: Vector-based similarity matching
- **Relevance Scoring**: Rank information by relevance

**Usage**:
```go
selectStrategy := strategy.NewSelectStrategy(memoryStore, vectorStore)
contextEngine.RegisterStrategy(selectStrategy)
```

### 3. Compress Strategy
**Purpose**: Reduce context size through summarization and pruning

**Features**:
- **Message Compression**: Summarize conversation history
- **Key Point Extraction**: Identify crucial information
- **Scratchpad Pruning**: Remove less important data
- **Token Management**: Stay within context limits
- **Compression Metrics**: Track compression ratios

**Usage**:
```go
compressStrategy := strategy.NewCompressStrategy(summarizer)
contextEngine.RegisterStrategy(compressStrategy)
```

### 4. Isolate Strategy
**Purpose**: Create separate contexts for different agents and tasks

**Features**:
- **Agent Isolation**: Separate context per agent
- **Task Isolation**: Isolated contexts for specific tasks
- **Sandbox Execution**: Safe code execution environments
- **Context Sharing**: Controlled information sharing
- **Cleanup Management**: Automatic context cleanup

**Usage**:
```go
isolateStrategy := strategy.NewIsolateStrategy()
contextEngine.RegisterStrategy(isolateStrategy)
```

### 5. Adaptive Strategy
**Purpose**: Intelligently combine strategies based on context analysis

**Features**:
- **Automatic Selection**: Choose best strategies for current situation
- **Rule-Based Logic**: Configurable decision rules
- **Context Analysis**: Analyze complexity, token count, agent count
- **Strategy Combination**: Apply multiple strategies in sequence
- **Learning Capability**: Adapt based on performance

**Usage**:
```go
adaptiveStrategy := strategy.NewAdaptiveStrategy(strategies)
contextEngine.RegisterStrategy(adaptiveStrategy)
```

### 6. Shared Context Module
**Purpose**: Enable multi-agent collaboration and coordination

**Features**:
- **Context Sharing**: Share context data between agents
- **Agent Coordination**: Task assignment and load balancing
- **Communication Channels**: Inter-agent messaging system
- **Global State Management**: Shared state across all agents
- **Event Broadcasting**: Real-time event distribution

**Usage**:
```go
// Create shared context
sharedContext := shared.NewInMemorySharedContext()

// Create coordinator
coordinator := shared.NewCoordinator(sharedContext, coordinatorConfig)

// Create communication manager
commManager := shared.NewCommunicationManager(sharedContext, commConfig)

// Register agents
coordinator.RegisterAgent(ctx, agentID, metadata)

// Share context between agents
sharedContext.ShareContext(ctx, "agent1", "agent2", data)
```

## 🚀 Quick Start

### 1. Create Context Engine

```go
import (
    "github.com/voocel/mas/context/engine"
    "github.com/voocel/mas/context/memory"
    "github.com/voocel/mas/context/strategy"
    "github.com/voocel/mas/context/shared"
)

// Create storage components
memoryStore := memory.NewInMemoryStore(1000)
vectorStore := memory.NewInMemoryVectorStore()
checkpointer := engine.NewInMemoryCheckpointer(100)

// Create context engine
contextEngine := engine.NewContextEngine(
    engine.WithMemory(memoryStore),
    engine.WithVectorStore(vectorStore),
    engine.WithCheckpointer(checkpointer),
)
```

### 2. Register Strategies

```go
// Create strategies
writeStrategy := strategy.NewWriteStrategy(memoryStore)
selectStrategy := strategy.NewSelectStrategy(memoryStore, vectorStore)
compressStrategy := strategy.NewCompressStrategy(summarizer)
isolateStrategy := strategy.NewIsolateStrategy()

// Register strategies
contextEngine.RegisterStrategy(writeStrategy)
contextEngine.RegisterStrategy(selectStrategy)
contextEngine.RegisterStrategy(compressStrategy)
contextEngine.RegisterStrategy(isolateStrategy)

// Create and register adaptive strategy
strategies := map[string]strategy.ContextStrategy{
    "write":    writeStrategy,
    "select":   selectStrategy,
    "compress": compressStrategy,
    "isolate":  isolateStrategy,
}
adaptiveStrategy := strategy.NewAdaptiveStrategy(strategies)
contextEngine.RegisterStrategy(adaptiveStrategy)
```

### 3. Use with Agents

```go
// Apply strategy to context state
state := contextpkg.NewContextState("thread_123", "agent_1")
updatedState, err := contextEngine.ApplyStrategy(ctx, "adaptive", state)
if err != nil {
    log.Fatal(err)
}

// Create checkpoint
err = contextEngine.CreateCheckpoint(ctx, updatedState)
if err != nil {
    log.Printf("Failed to create checkpoint: %v", err)
}
```

## 🔧 Configuration

### Write Strategy Configuration

```go
config := strategy.WriteConfig{
    EnableScratchpad:          true,
    EnableMemoryStorage:       true,
    MemoryImportanceThreshold: 0.7,
    MaxScratchpadSize:         1000,
    AutoSummarize:             true,
}
writeStrategy := strategy.NewWriteStrategy(memoryStore, config)
```

### Select Strategy Configuration

```go
config := strategy.SelectConfig{
    MaxMemories:        10,
    MaxTools:           5,
    MaxKnowledge:       8,
    RelevanceThreshold: 0.6,
    EnableSemanticSearch: true,
    MemoryDecayFactor:  0.1,
}
selectStrategy := strategy.NewSelectStrategy(memoryStore, vectorStore, config)
```

### Compress Strategy Configuration

```go
config := strategy.CompressConfig{
    MaxTokens:        4000,
    CompressionRatio: 0.3,
    PreserveRecent:   5,
    EnableSummary:    true,
    EnableKeyPoints:  true,
    MinImportance:    0.5,
}
compressStrategy := strategy.NewCompressStrategy(summarizer, config)
```

### Adaptive Strategy Configuration

```go
config := strategy.AdaptiveConfig{
    MaxTokens:            4000,
    CompressionThreshold: 0.8,
    IsolationThreshold:   3,
    SelectionThreshold:   0.6,
    EnableLearning:       true,
    MaxStrategies:        3,
}
adaptiveStrategy := strategy.NewAdaptiveStrategy(strategies, config)
```

## 📊 Monitoring and Metrics

### Memory Statistics

```go
memStats := memoryStore.GetStats()
fmt.Printf("Total memories: %d\n", memStats.TotalMemories)
fmt.Printf("Memory types: %+v\n", memStats.TypeCounts)
```

### Context Analysis

```go
analysis := contextEngine.AnalyzeState(state)
fmt.Printf("Token count: %d\n", analysis.TokenCount)
fmt.Printf("Complexity: %.2f\n", analysis.Complexity)
fmt.Printf("Memory pressure: %.2f\n", analysis.MemoryPressure)
```

## 🔄 Integration with Workflow

The context engineering module integrates seamlessly with the workflow module:

```go
// In agent implementation
func (a *Agent) ChatWithCommand(ctx context.Context, state *contextpkg.ContextState) (*workflow.AgentCommand, error) {
    // Process with context engine
    // ...
    
    return workflow.NewCommand().
        UpdateMessages(response).
        WithStrategy("adaptive").  // Apply adaptive strategy
        HandoffTo("next_agent").
        Build(), nil
}
```

## 🎯 Best Practices

1. **Use Adaptive Strategy**: Let the system choose the best strategies automatically
2. **Configure Thresholds**: Tune thresholds based on your use case
3. **Monitor Memory**: Keep track of memory usage and cleanup
4. **Checkpoint Regularly**: Save important states for recovery
5. **Measure Performance**: Monitor token usage and compression ratios

## 🔮 Future Enhancements

- **LLM-based Summarization**: Use LLMs for better summarization
- **Vector Embeddings**: Implement proper vector similarity search
- **Distributed Storage**: Support for distributed memory stores
- **Advanced Analytics**: ML-based context optimization
- **Real-time Adaptation**: Dynamic strategy adjustment based on performance

## 📚 Examples

See the `examples/context_engineering/` directory for complete examples:

- `basic_usage/`: Basic context engineering usage
- `shared_context/`: Multi-agent collaboration with shared context
- `memory_management/`: Advanced memory management
- `adaptive_strategies/`: Custom adaptive rules
