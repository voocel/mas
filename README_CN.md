# MAS

MAS 是一个现代的、高性能的 Go 多智能体系统框架，专为构建具有原生 Go 性能和类型安全的智能应用而设计。

![架构图](./docs/architecture.md)

[文档](./docs/) | [示例](./examples/) | [English](./README.md)

## 安装

```bash
go get github.com/voocel/mas
```

## 快速开始

**设置环境变量：**

```bash
export LLM_API_KEY="your-api-key"
export LLM_MODEL="gpt-5"
export LLM_BASE_URL="https://api.openai.com/v1"  # 可选
```

**基础用法：**

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
    // 创建 LLM 模型
    model, _ := llm.NewLiteLLMAdapter(
        os.Getenv("LLM_MODEL"),
        os.Getenv("LLM_API_KEY"),
        os.Getenv("LLM_BASE_URL"),
    )

    // 创建智能体
    agent := agent.NewAgent("assistant", "AI Assistant", model)

    // 执行对话
    ctx := runtime.NewContext(context.Background(), "session-1", "trace-1")
    response, _ := agent.Execute(ctx, schema.Message{
        Role:    schema.RoleUser,
        Content: "你好，你好吗？",
    })

    fmt.Printf("Agent: %s\n", response.Content)
}
```

**带工具的智能体：**

```go
import "github.com/voocel/mas/tools/builtin"

// 创建带计算器工具的智能体
calculator := builtin.NewCalculator()
agent := agent.NewAgent("math-agent", "Math Assistant", model,
    agent.WithTools(calculator),
)

response, _ := agent.Execute(ctx, schema.Message{
    Role:    schema.RoleUser,
    Content: "计算 15 * 8 + 7",
})
```

**多智能体协作：**

```go
import "github.com/voocel/mas/orchestrator"

// 创建编排器
orch := orchestrator.NewOrchestrator()

// 添加智能体
orch.AddAgent("researcher", researcher)
orch.AddAgent("writer", writer)

// 执行并自动路由
result, _ := orch.Execute(ctx, orchestrator.ExecuteRequest{
    Input: schema.Message{
        Role:    schema.RoleUser,
        Content: "研究并撰写关于AI趋势的文章",
    },
    Type: orchestrator.ExecuteTypeAuto, // 自动路由到最佳智能体
})
```

**工作流编排：**

```go
import "github.com/voocel/mas/workflows"

// 创建工作流步骤
preprocessStep := workflows.NewFunctionStep(
    workflows.NewStepConfig("preprocess", "数据预处理"),
    func(ctx runtime.Context, input schema.Message) (schema.Message, error) {
        // 处理逻辑
        return processedMessage, nil
    },
)

// 构建链式工作流
workflow := workflows.NewChainBuilder("data-pipeline", "数据处理管道").
    Then(preprocessStep).
    Then(analysisStep).
    Then(summaryStep).
    Build()

// 执行工作流
orch.AddWorkflow("data-pipeline", workflow)
result, _ := orch.Execute(ctx, orchestrator.ExecuteRequest{
    Input:  inputMessage,
    Target: "data-pipeline",
    Type:   orchestrator.ExecuteTypeWorkflow,
})
```

## 核心概念

### Agent vs Workflow

**Agent（智能体）** - 自主的智能实体，具有：

- **智能决策**: 基于LLM的推理和工具选择
- **动态行为**: 适应不同场景和上下文
- **工具集成**: 访问各种工具和能力
- **记忆管理**: 维护对话历史和上下文

**Workflow（工作流）** - 结构化的流程编排，具有：

- **确定性执行**: 预定义的步骤和控制流
- **并行处理**: 独立任务的并发执行
- **条件分支**: 基于条件的动态路由
- **可组合性**: 步骤可以是函数、智能体或其他工作流

**灵活组合**：

- **Agent in Workflow**: 工作流步骤可以是智能体
- **Workflow in Agent**: 智能体可以调用工作流作为工具
- **Hybrid Execution**: 在同一任务中混合使用智能体和工作流

## 示例与集成


## 许可证

Apache License 2.0 License
