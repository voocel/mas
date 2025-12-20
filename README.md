# MAS

MAS is a lightweight, pluggable Go multi‑agent SDK focused on simplicity and composability.

- **Lightweight**: an Agent is just a description (prompt + tools)
- **Pluggable**: the Runner owns the execution loop and can be customized
- **Easy to start**: 3–5 lines to run a single agent


[Examples](./examples/) | [Chinese](./README_CN.md)

## Install

```bash
go get github.com/voocel/mas
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/voocel/mas"
    "github.com/voocel/mas/agent"
    "github.com/voocel/mas/llm"
    "github.com/voocel/mas/runner"
    "github.com/voocel/mas/schema"
    "github.com/voocel/mas/tools/builtin"
)

func main() {
    model := llm.NewOpenAIModel(
        "gpt-5",
        os.Getenv("OPENAI_API_KEY"),
        os.Getenv("OPENAI_API_BASE_URL"),
    )

    // Minimal entry (recommended)
    resp, err := mas.Query(
        context.Background(),
        model,
        "Compute 15 * 8 + 7",
        mas.WithPreset("assistant"),
        mas.WithTools(builtin.NewCalculator()),
    )
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Println(resp.Content)

    // Advanced: custom Runner
    ag := agent.New(
        "assistant",
        "assistant",
        agent.WithSystemPrompt("You are a helpful assistant."),
        agent.WithTools(builtin.NewCalculator()),
    )

    r := runner.New(runner.Config{Model: model})

    resp, err := r.Run(context.Background(), ag, schema.Message{
        Role:    schema.RoleUser,
        Content: "Compute 15 * 8 + 7",
    })
    if err != nil {
        fmt.Println("error:", err)
        return
    }

    fmt.Println(resp.Content)
}
```

## Session Client

```go
cli, _ := mas.NewClient(
    model,
    mas.WithPreset("assistant"),
    mas.WithTools(builtin.NewCalculator()),
)
resp, _ := cli.Send(context.Background(), "Continue with 9 * 9")
```

## Structured Output (JSON Schema)

```go
format := &llm.ResponseFormat{
    Type: "json_object",
}
resp, _ := mas.Query(
    context.Background(),
    model,
    "Return JSON {\"answer\": 42}",
    mas.WithResponseFormat(format),
)
```

## Full Result (Usage/Tool Trace)

```go
result, _ := mas.QueryWithResult(
    context.Background(),
    model,
    "Compute 6 * 7",
)
fmt.Println(result.Message.Content, result.Usage.TotalTokens)
```

## Multi‑Agent (Team)

```go
import "github.com/voocel/mas/multi"

team := multi.NewTeam()
team.Add("researcher", researcher)
team.Add("writer", writer)

ag, _ := team.Route("researcher")
resp, _ := runner.Run(ctx, ag, msg)
```

## Collaboration Modes (Light but Powerful)

```go
// Sequential collaboration
resp, _ := multi.RunSequential(ctx, r, []*agent.Agent{researcher, writer}, msg)

// Parallel collaboration + reduce
resp, _ := multi.RunParallel(ctx, r, []*agent.Agent{a1, a2}, msg, multi.FirstReducer)

// Dynamic routing (handoff)
router := &multi.KeywordRouter{
    Rules:   map[string]string{"stats": "analyst", "write": "writer"},
    Default: "assistant",
}
resp, _ := multi.RunHandoff(ctx, r, team, router, msg, multi.WithMaxSteps(3))
```

## Middleware & Policies

```go
import "github.com/voocel/mas/middleware"

r := runner.New(runner.Config{
    Model: model,
    Middlewares: []runner.Middleware{
        &middleware.TimeoutMiddleware{LLMTimeout: 10 * time.Second, ToolTimeout: 20 * time.Second},
        &middleware.RetryMiddleware{MaxAttempts: 3},
        middleware.NewToolAllowlist("calculator", "web_search"),
        middleware.NewToolCapabilityPolicy(
            middleware.AllowOnly(tools.CapabilityNetwork),
            middleware.Deny(tools.CapabilityFile),
        ),
    },
})
```

## Observability & Tracing

```go
import "github.com/voocel/mas/observer"

r := runner.New(runner.Config{
    Model:    model,
    Observer: observer.NewLoggerObserver(os.Stdout),
    Tracer:   observer.NewSimpleTimerTracer(os.Stdout),
})
```
Logs include `run_id`, `step_id`, and `span_id` for correlation.

## Structured Logs & Metrics

```go
import (
    "github.com/voocel/mas/middleware"
    "github.com/voocel/mas/observer"
)

metrics := &middleware.MetricsObserver{}
obs := observer.NewCompositeObserver(
    observer.NewJSONObserver(os.Stdout),
    metrics,
)
```

## Routing (Optional)

```go
router := &multi.KeywordRouter{
    Rules:   map[string]string{"stats": "analyst", "write": "writer"},
    Default: "assistant",
}
ag, _ := router.Select(msg, team)
```

## Core Concepts

- **Agent**: describes role, system prompt and tools
- **Runner**: drives the execution loop (LLM → tools → feedback)
- **Tool**: independent capability with optional side‑effect flags
- **Memory**: conversation store (in‑memory window by default)

## License

Apache License 2.0
