# MAS - 灵活的多智能体框架 | [English](README.md)

MAS (Multi-Agent System) 是一个用 Go 语言编写的灵活、模块化的多智能体框架，专为快速构建基于大语言模型（LLM）的智能应用而设计。

## 框架特点

- **开放式架构**：核心组件通过接口定义，支持自定义扩展和替换
- **LLM集成**：内置支持 OpenAI API，易于扩展支持其他模型
- **智能体系统**：提供基础智能体接口和实现，支持感知-思考-行动循环
- **工具集成**：灵活的工具定义和调用机制，让智能体能够调用外部功能
- **内存系统**：支持多种记忆类型，包括短期和长期记忆
- **知识图谱**：实体和关系的结构化知识表示，增强智能体的长期记忆和推理能力
- **通信机制**：智能体间的消息传递系统，支持点对点和广播通信
- **任务编排**：任务定义、分配和执行的统一管理

## 项目结构

```
mas/
├── agent/           # 智能体定义和实现
├── communication/   # 通信系统
├── knowledge/       # 知识图谱系统
├── llm/             # 大语言模型集成
├── memory/          # 记忆系统
├── orchestrator/    # 任务编排
├── tools/           # 工具系统
├── examples/        # 示例项目
└── go.mod
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
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/tools"
)

func main() {
	// 初始化LLM提供者
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
		30*time.Second,
	)

	// 创建智能体
	assistant := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:        "assistant",
		LLMProvider: provider,
		SystemPrompt: `你是一个有用的助手。`,
	})

	// 创建工具
	toolRegistry := tools.NewRegistry()
	// 注册工具...

	// 运行智能体
	result, err := assistant.Process(context.Background(), "你好，请介绍下自己")
	if err != nil {
		log.Fatalf("处理失败: %v", err)
	}

	fmt.Println(result)
}
```

## 简单示例项目

### 1. 聊天助手（Chat Assistant）

一个基础的多轮对话聊天助手，展示如何维护对话历史和状态。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/llm"
)

func main() {
	// 初始化LLM提供者
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4o",
		30*time.Second,
	)

	// 创建对话记忆
	conversationMemory := memory.NewConversationMemory(10) // 保存最近10轮对话
	
	// 创建聊天智能体
	chatbot := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:         "Chatbot",
		Provider:     provider,
		SystemPrompt: "你是一个友好的聊天助手，你会用简短、清晰的方式回答问题。",
		Memory:       conversationMemory,
		MaxTokens:    1000,
		Temperature:  0.7,
	})

	// 主对话循环
	fmt.Println("聊天助手已启动，输入 'exit' 退出")
	for {
		fmt.Print("> ")
		var input string
		fmt.Scanln(&input)
		
		if input == "exit" {
			break
		}
		
		response, err := chatbot.Process(context.Background(), input)
		if err != nil {
			log.Printf("处理失败: %v", err)
			continue
		}
		
		fmt.Println(response)
	}
}
```

运行示例：

```bash
cd examples/chat_assistant
export OPENAI_API_KEY="your_api_key"
go run main.go
```

### 2. 任务规划器（Task Planner）

展示如何创建一个简单的任务规划智能体系统，将复杂目标分解为可执行的步骤。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
)

func main() {
	// 初始化LLM提供者
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
		30*time.Second,
	)

	// 创建规划智能体
	planner := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Planner",
		Provider: provider,
		SystemPrompt: `你是一个任务规划专家。你的工作是将用户的目标分解为详细的步骤计划。
每个步骤应该具体、可操作，并具有逻辑顺序。`,
		MaxTokens:   2000,
		Temperature: 0.2,
	})

	// 创建执行智能体
	executor := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Executor",
		Provider: provider,
		SystemPrompt: `你是一个执行专家。你的工作是为规划好的每个步骤提供详细的执行建议。
对于每个步骤，提供具体的行动指南、可能的资源和注意事项。`,
		MaxTokens:   1500,
		Temperature: 0.3,
	})

	// 用户目标
	goal := "学习Go语言并用它开发一个Web应用"

	// 首先使用规划智能体生成计划
	fmt.Println("正在为目标生成计划:", goal)
	planResult, err := planner.Process(context.Background(), goal)
	if err != nil {
		log.Fatalf("规划失败: %v", err)
	}

	plan, ok := planResult.(string)
	if !ok {
		log.Fatalf("规划结果类型错误")
	}

	fmt.Println("\n===== 计划 =====")
	fmt.Println(plan)

	// 然后使用执行智能体提供执行建议
	executionRequest := fmt.Sprintf("为以下计划提供详细的执行建议:\n\n%s", plan)
	executionResult, err := executor.Process(context.Background(), executionRequest)
	if err != nil {
		log.Fatalf("执行建议生成失败: %v", err)
	}

	fmt.Println("\n===== 执行建议 =====")
	fmt.Println(executionResult)
}
```

运行示例：

```bash
cd examples/task_planner
export OPENAI_API_KEY="your_api_key"
go run main.go
```

### 3. 协作写作（Collaborative Writing）

展示如何创建一个协作写作系统，多个智能体协作完成一篇文章。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/communication"
	"github.com/voocel/mas/llm"
)

func main() {
	// 初始化LLM提供者
	provider := llm.NewOpenAIProvider(
		os.Getenv("OPENAI_API_KEY"),
		"gpt-4",
		30*time.Second,
	)

	// 创建通信总线
	bus := communication.NewInMemoryBus()

	// 创建编辑智能体
	editor := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Editor",
		Provider: provider,
		SystemPrompt: `你是一名资深编辑。你的任务是审查和完善作家提供的内容。
关注逻辑连贯性、结构、风格以及整体质量。提供建设性的修改建议。`,
		MaxTokens:   1500,
		Temperature: 0.3,
	})

	// 创建作家智能体
	writer := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Writer",
		Provider: provider,
		SystemPrompt: `你是一名有创意的作家。你的任务是根据主题创作原创内容。
注重内容的创意性、趣味性和表达力。`,
		MaxTokens:   2000,
		Temperature: 0.7,
	})

	// 创建润色智能体
	polisher := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:     "Polisher",
		Provider: provider,
		SystemPrompt: `你是一名文字润色专家。你的任务是对文稿进行最终润色。
提高语言的优美度，修正任何语法或标点错误，确保整体风格一致。`,
		MaxTokens:   1500,
		Temperature: 0.4,
	})

	// 开始协作写作过程
	topic := "人工智能在日常生活中的应用"
	ctx := context.Background()

	fmt.Println("开始协作写作，主题:", topic)
	
	// 第一步：作家创作初稿
	fmt.Println("\n1. 作家正在创作初稿...")
	initialDraft, err := writer.Process(ctx, fmt.Sprintf("请为主题'%s'创作一篇800字左右的文章", topic))
	if err != nil {
		log.Fatalf("创作初稿失败: %v", err)
	}
	
	fmt.Println("\n===== 初稿 =====")
	fmt.Println(initialDraft)
	
	// 第二步：编辑审查并提供修改建议
	fmt.Println("\n2. 编辑正在审查...")
	editorFeedback, err := editor.Process(ctx, fmt.Sprintf("请审查并提供修改建议：\n\n%s", initialDraft))
	if err != nil {
		log.Fatalf("编辑审查失败: %v", err)
	}
	
	fmt.Println("\n===== 编辑建议 =====")
	fmt.Println(editorFeedback)
	
	// 第三步：作家根据建议修改
	fmt.Println("\n3. 作家正在修改...")
	revisedDraft, err := writer.Process(ctx, fmt.Sprintf("请根据编辑的建议修改文章：\n\n原文：\n%s\n\n编辑建议：\n%s", initialDraft, editorFeedback))
	if err != nil {
		log.Fatalf("修改失败: %v", err)
	}
	
	fmt.Println("\n===== 修改稿 =====")
	fmt.Println(revisedDraft)
	
	// 第四步：最终润色
	fmt.Println("\n4. 润色专家正在进行最终润色...")
	finalDraft, err := polisher.Process(ctx, fmt.Sprintf("请对这篇文章进行最终润色：\n\n%s", revisedDraft))
	if err != nil {
		log.Fatalf("润色失败: %v", err)
	}
	
	fmt.Println("\n===== 最终稿 =====")
	fmt.Println(finalDraft)
	
	fmt.Println("\n协作写作完成！")
}
```

运行示例：

```bash
cd examples/collaborative_writing
export OPENAI_API_KEY="your_api_key"
go run main.go
```

### 知识图谱智能体

位于 `examples/knowledge_agent/` 的示例项目展示了如何集成知识图谱到智能体中，增强智能体的知识表示和推理能力。

该示例演示：
- 如何创建和使用知识图谱
- 如何定义实体和关系
- 如何在智能体中利用知识图谱提供信息
- 如何查询相关知识以增强回答质量

运行示例：

```bash
cd examples/knowledge_agent
export OPENAI_API_KEY="your_api_key"
go run main.go
```

## 扩展框架

MAS框架设计为高度可扩展，你可以：

1. **实现自定义智能体**：继承基础智能体接口，根据需求定制行为
2. **添加新工具**：实现工具接口，为智能体提供新的能力
3. **扩展LLM支持**：添加新的语言模型提供者
4. **自定义记忆模型**：实现记忆接口，提供不同类型的记忆系统
5. **增强知识图谱**：实现更复杂的图数据库后端或优化查询性能
6. **增强通信机制**：支持更复杂的智能体间通信模式
