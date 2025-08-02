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
- **工具系统**: 可扩展的工具框架，提供外部能力
- **内存管理**: 对话和摘要内存实现
- **团队协作**: 多智能体工作流和协调
- **LLM集成**: 基于 [litellm](https://github.com/voocel/litellm) 支持多个提供商
- **轻量级**: 最少依赖，易于嵌入
- **流式API**: 可链式调用的配置方法

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
import (
    "github.com/voocel/mas"
    "github.com/voocel/mas/tools"
    "github.com/voocel/mas/memory"
)

func main() {
    agent := mas.NewAgent("o3", os.Getenv("OPENAI_API_KEY")).
        WithTools(tools.Calculator(), tools.WebSearch()).
        WithMemory(memory.Conversation(10)).
        WithSystemPrompt("你是一个有用的研究助手。")
    
    response, err := agent.Chat(context.Background(), 
        "计算250的15%，然后搜索有关百分比的信息")
    // 智能体将自动使用计算器工具和网络搜索
}
```

### 团队协作

```go
func main() {
    // 创建专业化智能体
    researcher := mas.NewAgent("gemini-2.5-pro", apiKey).
        WithSystemPrompt("你是一个研究员。收集关键信息。").
        WithTools(tools.WebSearch())
    
    writer := mas.NewAgent("claude-4-sonnet", apiKey).
        WithSystemPrompt("你是一个作家。创建引人入胜的内容。")
    
    // 创建团队工作流
    team := mas.NewTeam().
        Add("researcher", researcher).
        Add("writer", writer).
        WithFlow("researcher", "writer")
    
    result, err := team.Execute(context.Background(), 
        "研究并撰写关于可再生能源好处的文章")
}
```

## 示例

[`examples/`](examples/) 目录包含全面的示例：

- **[基本用法](examples/basic/)** - 简单的智能体交互和配置
- **[工具用法](examples/tools/)** - 内置和自定义工具
- **[团队协作](examples/team/)** - 多智能体工作流

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
- **文件读写器** - 读写文件
- **目录列表器** - 浏览文件系统
- **文件信息** - 获取文件元数据

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

## 团队模式

### 顺序处理
```go
team := mas.NewTeam().
    Add("analyzer", analyzerAgent).
    Add("writer", writerAgent).
    Add("editor", editorAgent).
    WithFlow("analyzer", "writer", "editor")
```

### 并行处理
```go
team := mas.NewTeam().
    Add("tech", techAgent).
    Add("business", businessAgent).
    Add("risk", riskAgent).
    WithFlow("tech", "business", "risk").
    WithParallel(true)
```

### 共享内存
```go
sharedMemory := memory.ThreadSafe(memory.Conversation(30))

team := mas.NewTeam().
    Add("agent1", agent1).
    Add("agent2", agent2).
    SetSharedMemory(sharedMemory)
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
├── team.go            # 多智能体协作
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

## 贡献

欢迎贡献！请随时：

- 报告错误
- 建议功能
- 提交拉取请求
- 改进文档

## 许可证

MIT许可证 - 详见 [LICENSE](LICENSE) 文件。

## 相关项目

- [litellm](https://github.com/voocel/litellm) - 统一LLM客户端库
- [OpenAI Go](https://github.com/sashabaranov/go-openai) - OpenAI API客户端
