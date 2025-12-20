# MAS

MAS 是一个轻量级、可插拔的 Go 多 Agent SDK，强调“少概念、可组合、易上手”。

- **轻量**：Agent 只负责描述（系统提示、工具集），不隐藏执行逻辑
- **可插拔**：执行链路由 Runner 控制，可替换模型、工具执行器与中间件
- **易用**：3~5 行即可跑通单 Agent


[示例](./examples/) | [English](./README.md)

## 安装

```bash
go get github.com/voocel/mas
```

## 快速开始

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

    // 极简入口（推荐）
    resp, err := mas.Query(
        context.Background(),
        model,
        "计算 15 * 8 + 7",
        mas.WithPreset("assistant"),
        mas.WithTools(builtin.NewCalculator()),
    )
    if err != nil {
        fmt.Println("error:", err)
        return
    }
    fmt.Println(resp.Content)

    // 进阶：自定义 Runner
    ag := agent.New(
        "assistant",
        "assistant",
        agent.WithSystemPrompt("你是一个友好的助手，善于解释与计算。"),
        agent.WithTools(builtin.NewCalculator()),
    )

    r := runner.New(runner.Config{Model: model})

    resp, err := r.Run(context.Background(), ag, schema.Message{
        Role:    schema.RoleUser,
        Content: "计算 15 * 8 + 7",
    })
    if err != nil {
        fmt.Println("error:", err)
        return
    }

    fmt.Println(resp.Content)
}
```

## 会话式 Client

```go
cli, _ := mas.NewClient(
    model,
    mas.WithPreset("assistant"),
    mas.WithTools(builtin.NewCalculator()),
)
resp, _ := cli.Send(context.Background(), "继续计算 9 * 9")
```

## 结构化输出（JSON Schema）

```go
format := &llm.ResponseFormat{
    Type: "json_object",
}
resp, _ := mas.Query(
    context.Background(),
    model,
    "用 JSON 返回 {\"answer\": 42}",
    mas.WithResponseFormat(format),
)
```

## 完整结果（Usage/工具轨迹）

```go
result, _ := mas.QueryWithResult(
    context.Background(),
    model,
    "计算 6 * 7",
)
fmt.Println(result.Message.Content, result.Usage.TotalTokens)
```

## 多 Agent（轻量 Team）

```go
import "github.com/voocel/mas/multi"

team := multi.NewTeam()
team.Add("researcher", researcher)
team.Add("writer", writer)

ag, _ := team.Route("researcher")
resp, _ := runner.Run(ctx, ag, msg)
```

## 协作模式（轻量 but Powerful）

```go
// 顺序协作
resp, _ := multi.RunSequential(ctx, r, []*agent.Agent{researcher, writer}, msg)

// 并行协作 + 合并
resp, _ := multi.RunParallel(ctx, r, []*agent.Agent{a1, a2}, msg, multi.FirstReducer)

// 动态路由（handoff）
router := &multi.KeywordRouter{
    Rules:   map[string]string{"统计": "analyst", "写作": "writer"},
    Default: "assistant",
}
resp, _ := multi.RunHandoff(ctx, r, team, router, msg, multi.WithMaxSteps(3))
```

## 中间件与策略

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

## 观测与追踪

```go
import "github.com/voocel/mas/observer"

r := runner.New(runner.Config{
    Model:    model,
    Observer: observer.NewLoggerObserver(os.Stdout),
    Tracer:   observer.NewSimpleTimerTracer(os.Stdout),
})
```

## 结构化日志与指标

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

## 路由（可选）

```go
router := &multi.KeywordRouter{
    Rules:   map[string]string{"统计": "analyst", "写作": "writer"},
    Default: "assistant",
}
ag, _ := router.Select(msg, team)
```

## 核心概念

- **Agent**：仅描述角色、系统提示与工具集
- **Runner**：执行链路核心（模型调用 → 工具调用 → 回填 → 再生成）
- **Tool**：独立功能单元，可标记能力（network/file/unsafe）
- **Memory**：对话记忆（默认内存窗口）

## 设计理念

- **显式执行**：所有执行步骤可观察、可拦截、可替换
- **低心智负担**：最小 API 面向快速上手
- **可扩展**：中间件与工具执行器可以自由注入

## 许可证

Apache License 2.0
