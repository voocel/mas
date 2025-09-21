package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools/builtin"
)

func main() {
	fmt.Println(strings.Repeat("=", 50))

	model := llm.NewOpenAIModel("gpt-4.1-mini", os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_API_BASE_URL"))
	basicAgent := createBasicAgent(model)

	toolAgent := createToolAgent(model)

	demonstrateAgentConversation(basicAgent)

	demonstrateToolUsage(toolAgent)

}

func createBasicAgent(model llm.ChatModel) *agent.BaseAgent {
	return agent.NewAgent(
		"basic_assistant",
		"basic_assistant",
		model,
		agent.WithSystemPrompt("You are a friendly AI assistant who can answer a wide range of questions and provide help."),
		agent.WithCapabilities(&agent.AgentCapabilities{
			CoreCapabilities: []agent.Capability{agent.CapabilityReasoning},
			Description:      "General AI assistant, suitable for basic conversations and Q&A",
			ComplexityLevel:  3,
			ConcurrencyLevel: 1,
		}),
	)
}

func createToolAgent(model llm.ChatModel) *agent.BaseAgent {
	calculator := builtin.NewCalculator()
	return agent.NewAgent(
		"tool_assistant",
		"tool_assistant",
		model,
		agent.WithSystemPrompt("You are an AI assistant that can use tools to perform calculations and other operations."),
		agent.WithTools(calculator),
		agent.WithCapabilities(&agent.AgentCapabilities{
			CoreCapabilities: []agent.Capability{
				agent.CapabilityToolUse,
				agent.CapabilityReasoning,
			},
			ToolTypes:        []string{"calculator", "math"},
			Description:      "AI assistant with tool usage capability",
			ComplexityLevel:  5,
			ConcurrencyLevel: 1,
		}),
	)
}

func createExpertAgent(model llm.ChatModel) *agent.BaseAgent {
	return agent.NewAgent(
		"data_analyst",
		"data_analyst",
		model,
		agent.WithSystemPrompt("You are a professional data analyst, skilled in data processing, statistical analysis, and insight discovery."),
		agent.WithCapabilities(&agent.AgentCapabilities{
			CoreCapabilities: []agent.Capability{
				agent.CapabilityAnalysis,
				agent.CapabilityToolUse,
				agent.CapabilityReasoning,
			},
			Expertise:        []string{"data analysis", "statistics", "Business Intelligence"},
			ToolTypes:        []string{"analytics", "statistics"},
			Description:      "Professional data analyst, skilled in data insights and business analysis",
			ComplexityLevel:  8,
			ConcurrencyLevel: 2,
		}),
	)
}

func demonstrateAgentConversation(ag *agent.BaseAgent) {
	ctx := runtime.NewContext(context.Background(), "demo", "conversation")

	message := schema.Message{
		Role:    schema.RoleUser,
		Content: "hi",
	}

	response, err := ag.Execute(ctx, message)
	if err != nil {
		return
	}

	fmt.Printf("user: %s\n", message.Content)
	fmt.Printf("AI %s: %s\n", ag.Name(), response.Content)
}

func demonstrateToolUsage(ag *agent.BaseAgent) {
	ctx := runtime.NewContext(context.Background(), "demo", "tool_usage")

	message := schema.Message{
		Role:    schema.RoleUser,
		Content: "cal 2 + 3 * 4",
	}

	response, err := ag.Execute(ctx, message)
	if err != nil {
		return
	}

	fmt.Printf("user: %s\n", message.Content)
	fmt.Printf("AI %s: %s\n", ag.Name(), response.Content)

	tools := ag.Tools()
	for _, tool := range tools {
		fmt.Printf("   - %s: %s\n", tool.Name(), tool.Description())
	}
}

