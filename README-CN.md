# MAS - 轻量级Go多智能体框架

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![GoDoc](https://pkg.go.dev/badge/github.com/voocel/mas.svg)](https://pkg.go.dev/github.com/voocel/mas)

[English](README.md) | 中文

MAS (Multi-Agent System) 是一个轻量级、优雅的Go多智能体框架，旨在让开发者轻松地将智能体能力集成到应用程序中。

## 设计理念

- **简单至上**: 最小化API设计，零配置默认
- **易于集成**: 单个导入，无架构约束  
- **约定大于配置**: 合理默认值，最少设置要求
- **渐进式复杂度**: 支持从简单到复杂的用例

## 特性

- **智能代理**: 由LLM驱动的具有记忆和工具的智能体
- **工作流编排**: 基于状态图的多智能体协调
- **条件路由**: 基于上下文状态的动态工作流路径
- **人工干预**: 交互式审批和输入机制
- **工具系统**: 可扩展的工具框架，支持沙箱安全
- **内存管理**: 对话和摘要内存实现
- **LLM集成**: 基于 [litellm](https://github.com/voocel/litellm) 支持多个提供商
- **检查点与恢复**: 高级工作流持久化和恢复系统
  - **自动检查点**: 在关键点自动保存状态
  - **智能恢复**: 从中断点继续执行
  - **多种存储选项**: 文件、内存、数据库
  - **压缩支持**: 高效存储
  - **错误处理**: 优雅的故障恢复
- **轻量级**: 最少依赖，易于嵌入
- **流式API**: 可链式调用的配置方法

## 架构

```
mas/
├── agent.go            # 核心Agent实现，LLM集成
├── workflow.go         # 工作流编排和状态管理
├── tool.go             # 工具框架和接口
├── memory.go           # 内存系统（对话、摘要）
├── checkpoint.go       # 检查点接口和工具
├── types.go            # 核心类型定义和接口
├── errors.go           # 错误类型和处理
├── agent/              # Agent实现细节
│   ├── agent.go          # 核心Agent实现
│   ├── execution.go      # 工具调用执行逻辑
│   └── config.go         # 配置管理
├── workflow/           # 工作流执行引擎
│   ├── builder.go        # WorkflowBuilder实现
│   ├── context.go        # WorkflowContext实现
│   ├── executor.go       # 执行引擎
│   ├── nodes.go          # 所有节点类型实现
│   └── routing.go        # 条件路由逻辑
├── llm/                # LLM提供商抽象
│   ├── provider.go     # 提供商接口和工厂
│   ├── litellm.go      # LiteLLM适配器实现
│   ├── types.go        # LLM特定类型
│   └── converter.go    # 格式转换工具
├── memory/             # 内存实现
│   ├── conversation.go # 对话内存
│   ├── summary.go      # 带压缩的摘要内存
│   └── config.go       # 内存配置
├── checkpoint/         # 检查点系统
│   ├── manager.go      # 检查点管理器
│   ├── store.go        # 存储接口
│   ├── file.go         # 文件存储后端
│   ├── memory.go       # 内存存储
│   ├── redis.go        # Redis存储 (+build redis)
│   └── sqlite.go       # SQLite存储 (+build sqlite)
├── tools/              # 内置工具生态系统
├── examples/           # 使用示例
│   ├── basic/          # 基础智能体使用
│   ├── workflow/       # 多智能体工作流
│   ├── tools/          # 自定义工具和多工具使用
│   ├── baseurl/        # 自定义API端点
│   ├── checkpoint/     # 检查点和恢复
│   └── verify/         # 安装验证
└── internal/           # 内部工具
```

## 快速开始

### 安装

```bash
go get github.com/voocel/mas
```

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/voocel/mas"
)

func main() {
    // 创建一个最小配置的智能体
    agent := mas.NewAgent("o3", os.Getenv("OPENAI_API_KEY"))
    
    // 与智能体对话
    response, err := agent.Chat(context.Background(), "你好！你怎么样？")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(response)
}
```

### 使用工具和内存

```go
func main() {
    // 创建自定义工具
    greetingTool := mas.NewSimpleTool("greeting", "生成问候语", 
        func(ctx context.Context, params map[string]any) (any, error) {
            return "你好，世界！", nil
        })

    agent := mas.NewAgent("gpt-4", os.Getenv("OPENAI_API_KEY")).
        WithTools(greetingTool).
        WithMemory(mas.NewConversationMemory(10)).
        WithSystemPrompt("你是一个有用的研究助手。")
    
    response, _ := agent.Chat(context.Background(), "使用问候工具")
    fmt.Println(response)
}
```

### 自定义Base URL (DeepSeek、Ollama、Azure OpenAI等)

```go
func main() {
    // 使用自定义OpenAI兼容API
    config := mas.AgentConfig{
        Name:        "DeepSeekAgent",
        Model:       "deepseek-chat",
        APIKey:      os.Getenv("DEEPSEEK_API_KEY"),
        BaseURL:     "https://api.deepseek.com/v1",  // 自定义端点
        Temperature: 0.7,
        MaxTokens:   1000,
    }
    
    agent := mas.NewAgentWithConfig(config)
    response, _ := agent.Chat(context.Background(), "你好！")
    fmt.Println(response)
}
```

**支持的自定义端点：**
- DeepSeek: `https://api.deepseek.com/v1`
- 本地Ollama: `http://localhost:11434/v1`
- Azure OpenAI: `https://your-resource.openai.azure.com/openai/deployments/gpt-4`
- 任何OpenAI兼容的API

## 示例

运行示例来查看MAS的实际应用：

```bash
# 基础智能体使用
cd examples/basic && go run main.go

# 多智能体工作流
cd examples/workflow && go run main.go

# 自定义工具和多工具使用
cd examples/tools && go run main.go

# 自定义base URL配置
cd examples/baseurl && go run main.go

# 验证框架安装
cd examples/verify && go run main.go
```

### 文件工具沙箱

```go
// 无限制访问
tools.FileReader()

// 仅当前目录
sandbox := tools.DefaultSandbox()
tools.FileReaderWithSandbox(sandbox)

// 自定义允许路径
sandbox := &tools.FileSandbox{
    AllowedPaths: []string{"./data", "./uploads"},
    AllowCurrentDir: false,
}
tools.FileWriterWithSandbox(sandbox)
```

### 多智能体工作流

```go
func main() {
    // 创建专业化智能体
    researcher := mas.NewAgent("gemini-2.5-pro", apiKey).
        WithSystemPrompt("你是一个研究员。")

    writer := mas.NewAgent("claude-4-sonnet", apiKey).
        WithSystemPrompt("你是一个作家。")

    // 创建状态图工作流
    workflow := mas.NewWorkflow().
        AddNode(mas.NewAgentNode("researcher", researcher)).
        AddNode(mas.NewAgentNode("writer", writer)).
        AddEdge("researcher", "writer").
        SetStart("researcher")

    state, err := workflow.Execute(context.Background(), initialState)
}
```

### 条件路由

```go
// 简单条件路由
workflow.AddConditionalRoute("classifier",
    func(ctx *mas.WorkflowContext) bool {
        output := ctx.Get("output")
        return strings.Contains(fmt.Sprintf("%v", output), "技术")
    },
    "tech_expert", "biz_expert")

// 多分支条件
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

### 人工干预

```go
// 控制台输入，支持超时和验证
humanProvider := mas.NewConsoleInputProvider()
humanNode := mas.NewHumanNode("reviewer", "请审核内容：", humanProvider).
    WithOptions(
        mas.WithTimeout(5*time.Minute),
        mas.WithValidator(func(input string) error {
            if len(input) < 10 {
                return errors.New("反馈太短")
            }
            return nil
        }),
    )

// 自定义输入提供者（Web、API等）
type WebInputProvider struct{}
func (p *WebInputProvider) RequestInput(ctx context.Context, prompt string, options ...mas.HumanInputOption) (*mas.HumanInput, error) {
    // 实现基于Web的输入收集
}
```

## 示例

[`examples/`](examples/) 目录包含全面的示例：

- **[基本用法](examples/basic/)** - 简单的智能体交互和配置
- **[工具用法](examples/tools/)** - 内置和自定义工具，支持沙箱
- **[工作流编排](examples/workflow/)** - 多智能体工作流和协调

运行示例：

```bash
cd examples/basic
export OPENAI_API_KEY="your-api-key"
go run main.go
```

## 内置工具

MAS 开箱即用地提供有用的工具：

### 数学与计算
- **计算器** - 基本算术运算
- **高级计算器** - 表达式求值

### 网络
- **网络搜索** - 搜索引擎集成
- **HTTP请求** - REST API调用
- **网页抓取** - 提取网页内容
- **域名信息** - WHOIS、DNS、SSL信息

### 文件操作
- **文件读写器** - 读写文件，支持沙箱限制
- **目录列表器** - 浏览文件系统，支持路径限制
- **文件信息** - 获取文件元数据，支持访问控制

### 数据处理
- **JSON解析器** - 解析和操作JSON
- **URL缩短器** - 创建短链接

## 内存系统

### 对话内存
```go
// 记住最近10条消息
memory := memory.Conversation(10)

// 自定义配置
memory := memory.ConversationWithConfig(mas.MemoryConfig{
    MaxMessages: 50,
    TTL: 24 * time.Hour,
})
```

### 持久化内存
```go
// 自动保存到磁盘
memory := memory.Persistent(100, "./chat_history.json")
```

### 高级内存
```go
// 线程安全的共享内存
shared := memory.ThreadSafe(memory.Conversation(20))

// 多层级快慢存储
multiTier := memory.MultiTier(
    memory.Conversation(10),    // 快速内存
    memory.Persistent(1000, "./history.json"), // 慢速内存
    10, // 快速内存限制
)
```



## 自定义工具

轻松创建自己的工具：

```go
func MyCustomTool() mas.Tool {
    schema := &mas.ToolSchema{
        Type: "object",
        Properties: map[string]*mas.PropertySchema{
            "input": mas.StringProperty("要处理的输入文本"),
        },
        Required: []string{"input"},
    }
    
    return mas.NewTool(
        "my_tool",
        "此工具的功能描述",
        schema,
        func(ctx context.Context, params map[string]any) (any, error) {
            input := params["input"].(string)
            // 你的工具逻辑在这里
            return map[string]any{
                "result": "已处理: " + input,
            }, nil
        },
    )
}
```

## 架构

```
mas/
├── agent.go           # 核心智能体实现
├── tool.go            # 工具接口和注册表
├── memory.go          # 内存接口
├── workflow.go        # 多智能体工作流编排
├── llm/               # LLM提供商集成
├── tools/             # 内置工具
├── memory/            # 内存实现
└── examples/          # 使用示例
```

## LLM支持

基于 [litellm](https://github.com/voocel/litellm) 提供广泛的LLM提供商支持：

- **OpenAI** - o3, o4-mini, GPT-4.5, GPT-4.1, GPT-4o
- **Anthropic** - Claude 4 Opus, Claude 4 Sonnet, Claude 3.7 Sonnet
- **Google** - Gemini 2.5 Pro, Gemini 2.5 Flash
- **以及更多...**

```go
// 自动提供商检测
agent := mas.NewAgent("o3", apiKey)
agent := mas.NewAgent("claude-4-sonnet", apiKey)
agent := mas.NewAgent("gemini-2.5-pro", apiKey)
```

## 配置

### 智能体配置
```go
config := mas.AgentConfig{
    Name:         "MyAgent",
    Model:        "o3",
    APIKey:       apiKey,
    SystemPrompt: "你是一个有用的助手",
    Temperature:  0.7,
    MaxTokens:    2000,
    Tools:        []mas.Tool{tools.Calculator()},
    Memory:       memory.Conversation(10),
}

agent := mas.NewAgentWithConfig(config)
```

### 流式配置
```go
agent := mas.NewAgent("o3", apiKey).
    WithSystemPrompt("你是一个专家...").
    WithTemperature(0.3).
    WithMaxTokens(1000).
    WithTools(tools.Calculator()).
    WithMemory(memory.Conversation(5))
```

## 测试

```bash
go test ./...
```

## 文档

- [设计文档](CLAUDE.md) - 架构和设计决策
- [API参考](https://pkg.go.dev/github.com/voocel/mas) - 完整API文档
- [示例](examples/) - 实际使用示例

## 路线图

### 高优先级
- [x] **条件路由** - 基于上下文状态的动态工作流路径
- [x] **人工干预** - 交互式审批和输入机制
- [ ] **循环检测控制** - 智能循环处理和循环预防

### 中优先级
- [ ] **检查点恢复** - 工作流状态持久化和恢复
- [ ] **基于角色的智能体** - 内置角色系统和预定义行为
- [ ] **高级工具集成** - 工具链和条件工具使用

### 低优先级
- [ ] **监控与可观测性** - 内置追踪和指标
- [ ] **可视化工作流设计器** - 基于Web的工作流构建器
- [ ] **云集成** - 主流云平台的原生支持

## 贡献

欢迎贡献！请随时：

- 报告错误
- 建议功能
- 提交拉取请求
- 改进文档

## 许可证

Apache License 2.0 - 详见 [LICENSE](LICENSE) 文件。

## 相关项目

- [litellm](https://github.com/voocel/litellm) - 统一LLM客户端库
- [OpenAI Go](https://github.com/sashabaranov/go-openai) - OpenAI API客户端
