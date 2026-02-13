# AgentCore

**AgentCore** is a minimal, composable Go library for building AI agent applications.

[Examples](./examples/) | [简体中文](./README_CN.md)

## Install

```bash
go get github.com/voocel/agentcore
```

## Architecture

```
agentcore/            Agent core (types, loop, agent, events, subagent)
agentcore/llm/        LLM adapters (OpenAI, Anthropic, Gemini via litellm)
agentcore/tools/      Built-in tools: read, write, edit, bash
agentcore/memory/     Context compaction — auto-summarize long conversations
```

Core design:

- **Pure function loop** (`loop.go`) — double loop: inner processes tool calls + steering, outer handles follow-up
- **Stateful Agent** (`agent.go`) — consumes loop events to update state, same as any external listener
- **Event stream** — single `<-chan Event` output drives any UI (TUI, Web, Slack, logging)
- **Two-stage pipeline** — `TransformContext` (prune/inject) → `ConvertToLLM` (filter to LLM messages)
- **SubAgent tool** (`subagent.go`) — multi-agent via tool invocation, three modes: single, parallel, chain
- **Context compaction** (`memory/`) — automatic summarization when context approaches window limit

## Quick Start

### Single Agent

```go
package main

import (
    "fmt"
    "os"

    "github.com/voocel/agentcore"
    "github.com/voocel/agentcore/llm"
    "github.com/voocel/agentcore/tools"
)

func main() {
    model := llm.NewOpenAIModel("gpt-4.1-mini", os.Getenv("OPENAI_API_KEY"))

    agent := agentcore.NewAgent(
        agentcore.WithModel(model),
        agentcore.WithSystemPrompt("You are a helpful coding assistant."),
        agentcore.WithTools(
            tools.NewRead(),
            tools.NewWrite(),
            tools.NewEdit(),
            tools.NewBash("."),
        ),
    )

    agent.Subscribe(func(ev agentcore.Event) {
        if ev.Type == agentcore.EventMessageEnd {
            if msg, ok := ev.Message.(agentcore.Message); ok && msg.Role == agentcore.RoleAssistant {
                fmt.Println(msg.Content)
            }
        }
    })

    agent.Prompt("List the files in the current directory.")
    agent.WaitForIdle()
}
```

### Multi-Agent (SubAgent Tool)

Sub-agents are invoked as regular tools with isolated contexts:

```go
scout := agentcore.SubAgentConfig{
    Name:         "scout",
    Description:  "Fast codebase reconnaissance",
    Model:        llm.NewOpenAIModel("gpt-4.1-mini", apiKey),
    SystemPrompt: "Quickly explore and report findings. Be concise.",
    Tools:        []agentcore.Tool{tools.NewRead(), tools.NewBash(".")},
    MaxTurns:     5,
}

worker := agentcore.SubAgentConfig{
    Name:         "worker",
    Description:  "General-purpose executor",
    Model:        llm.NewOpenAIModel("gpt-4.1-mini", apiKey),
    SystemPrompt: "Implement tasks given to you.",
    Tools:        []agentcore.Tool{tools.NewRead(), tools.NewWrite(), tools.NewEdit(), tools.NewBash(".")},
}

agent := agentcore.NewAgent(
    agentcore.WithModel(model),
    agentcore.WithTools(agentcore.NewSubAgentTool(scout, worker)),
)
```

Three execution modes via tool call:

```jsonc
// Single: one agent, one task
{"agent": "scout", "task": "Find all API endpoints"}

// Parallel: concurrent execution
{"tasks": [{"agent": "scout", "task": "Find auth code"}, {"agent": "scout", "task": "Find DB schema"}]}

// Chain: sequential with {previous} context passing
{"chain": [{"agent": "scout", "task": "Find auth code"}, {"agent": "worker", "task": "Refactor based on: {previous}"}]}
```

### Steering & Follow-Up

```go
// Interrupt mid-run (delivered after current tool, remaining tools skipped)
agent.Steer(agentcore.Message{Role: agentcore.RoleUser, Content: "Stop and focus on tests instead."})

// Queue for after the agent finishes
agent.FollowUp(agentcore.Message{Role: agentcore.RoleUser, Content: "Now run the tests."})

// Cancel immediately
agent.Abort()
```

### Event Stream

All lifecycle events flow through a single channel — subscribe to drive any UI:

```go
agent.Subscribe(func(ev agentcore.Event) {
    switch ev.Type {
    case agentcore.EventMessageStart:    // assistant starts streaming
    case agentcore.EventMessageUpdate:   // streaming token delta
    case agentcore.EventMessageEnd:      // message complete
    case agentcore.EventToolExecStart:   // tool execution begins
    case agentcore.EventToolExecEnd:     // tool execution ends
    case agentcore.EventError:           // error occurred
    }
})
```

### Custom LLM (StreamFn)

Swap the LLM call with a proxy, mock, or custom implementation:

```go
agent := agentcore.NewAgent(
    agentcore.WithStreamFn(func(ctx context.Context, req *agentcore.LLMRequest) (*agentcore.LLMResponse, error) {
        // Route to your own proxy/gateway
        return callMyProxy(ctx, req)
    }),
)
```

### Context Compaction

Auto-summarize conversation history when approaching the context window limit. Hooks in via `TransformContext` — zero changes to core:

```go
import "github.com/voocel/agentcore/memory"

agent := agentcore.NewAgent(
    agentcore.WithModel(model),
    agentcore.WithTransformContext(memory.NewCompaction(memory.CompactionConfig{
        Model:         model,
        ContextWindow: 128000,
    })),
    agentcore.WithConvertToLLM(memory.CompactionConvertToLLM),
)
```

On each LLM call, compaction checks total tokens. When they exceed `ContextWindow - ReserveTokens` (default 16384), it:

1. Keeps recent messages (default 20000 tokens)
2. Summarizes older messages via LLM into a structured checkpoint (Goal / Progress / Key Decisions / Next Steps)
3. Tracks file operations (read/write/edit paths) across compacted messages
4. Supports incremental updates — subsequent compactions update the existing summary rather than re-summarizing

### Context Pipeline

```go
agent := agentcore.NewAgent(
    // Stage 1: prune old messages, inject external context
    agentcore.WithTransformContext(func(ctx context.Context, msgs []agentcore.AgentMessage) ([]agentcore.AgentMessage, error) {
        if len(msgs) > 100 {
            msgs = msgs[len(msgs)-50:]
        }
        return msgs, nil
    }),
    // Stage 2: filter to LLM-compatible messages
    agentcore.WithConvertToLLM(func(msgs []agentcore.AgentMessage) []agentcore.Message {
        var out []agentcore.Message
        for _, m := range msgs {
            if msg, ok := m.(agentcore.Message); ok {
                out = append(out, msg)
            }
        }
        return out
    }),
)
```

## Built-in Tools

| Tool | Description |
|------|-------------|
| `read` | Read file contents with head truncation (2000 lines / 50KB) |
| `write` | Write file with auto-mkdir |
| `edit` | Exact text replacement with fuzzy match, BOM/line-ending normalization, unified diff output |
| `bash` | Execute shell commands with tail truncation (2000 lines / 50KB) |

## API Reference

### Agent

| Method | Description |
|--------|-------------|
| `NewAgent(opts...)` | Create agent with options |
| `Prompt(input)` | Start new conversation turn |
| `Continue()` | Resume from current context |
| `Steer(msg)` | Inject steering message mid-run |
| `FollowUp(msg)` | Queue message for after completion |
| `Abort()` | Cancel current execution |
| `WaitForIdle()` | Block until agent finishes |
| `Subscribe(fn)` | Register event listener |
| `State()` | Snapshot of current state |

### Options

| Option | Description |
|--------|-------------|
| `WithModel(m)` | Set LLM model |
| `WithSystemPrompt(s)` | Set system prompt |
| `WithTools(t...)` | Set tool list |
| `WithMaxTurns(n)` | Safety limit (default: 10) |
| `WithStreamFn(fn)` | Custom LLM call function |
| `WithTransformContext(fn)` | Context transform (stage 1) |
| `WithConvertToLLM(fn)` | Message conversion (stage 2) |
| `WithSteeringMode(m)` | Queue drain mode: `"all"` or `"one-at-a-time"` |
| `WithFollowUpMode(m)` | Queue drain mode: `"all"` or `"one-at-a-time"` |

## License

Apache License 2.0
