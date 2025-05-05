package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/voocel/mas/knowledge"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/memory"
	"github.com/voocel/mas/tools"

	"github.com/google/uuid"
)

// LLMAgentConfig contains configuration parameters for creating an LLM agent
type LLMAgentConfig struct {
	ID           string
	Name         string
	Description  string
	MemoryConfig memory.Config
	Tools        []tools.Tool
	Provider     llm.Provider
	SystemPrompt string
	MaxTokens    int
	Temperature  float64
	Knowledge    knowledge.Graph
}

// LLMAgent represents an agent based on a large language model
type LLMAgent struct {
	BaseAgent
	provider       llm.Provider
	systemPrompt   string
	maxTokens      int
	temperature    float64
	state          map[string]interface{}
	stateMu        sync.RWMutex
	currentInput   interface{}
	currentThought string
}

// NewLLMAgent creates a new LLM agent
func NewLLMAgent(config LLMAgentConfig) *LLMAgent {
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	baseAgent := NewBaseAgent(config.Name)

	// Create memory system
	mem := memory.New(config.MemoryConfig)

	// Set up tools and knowledge graph
	baseAgent.tools = config.Tools
	baseAgent.knowledge = config.Knowledge
	baseAgent.memory = mem

	agent := &LLMAgent{
		BaseAgent:    *baseAgent,
		provider:     config.Provider,
		systemPrompt: config.SystemPrompt,
		maxTokens:    config.MaxTokens,
		temperature:  config.Temperature,
		state:        make(map[string]interface{}),
	}

	return agent
}

// Perceive handles input information in the perception phase
func (a *LLMAgent) Perceive(ctx context.Context, input interface{}) error {
	// Store current input
	a.currentInput = input

	// Add input to memory
	if a.memory != nil {
		err := a.memory.Add(ctx, memory.MemoryItem{
			ID:        uuid.New().String(),
			Content:   input,
			Type:      memory.TypeObservation,
			CreatedAt: time.Now(),
			Metadata:  map[string]interface{}{"source": "input"},
		})
		if err != nil {
			return fmt.Errorf("failed to add input to memory: %w", err)
		}
	}

	return nil
}

// Think processes information in the thinking phase
func (a *LLMAgent) Think(ctx context.Context) error {
	// Prepare prompt
	prompt := a.preparePrompt()

	log.Printf("Agent[%s] starting thinking phase", a.Name())

	// Call LLM for thinking
	log.Printf("Agent[%s] sending request to %s: model=%s", a.Name(), a.provider.ID(), a.GetModelName())
	log.Printf("Sending request...")

	resp, err := a.provider.ChatCompletion(ctx, llm.ChatCompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: a.temperature,
		MaxTokens:   a.maxTokens,
		Extra: map[string]interface{}{
			"agent_name": a.Name(),
		},
	})
	if err != nil {
		log.Printf("Agent[%s] request failed: %v", a.Name(), err)
		return fmt.Errorf("LLM call failed: %w", err)
	}

	log.Printf("Response received: status code=%d", 200) // Assuming status code is 200, may need to get from return value
	log.Printf("Response data received")

	// Extract thinking result
	if len(resp.Choices) > 0 {
		a.currentThought = resp.Choices[0].Message.Content
		log.Printf("Agent[%s] successfully parsed response: %s", a.Name(), truncateString(a.currentThought, 100))
	} else {
		log.Printf("Agent[%s] response has no choice results", a.Name())
	}

	// Add thinking result to memory
	if a.memory != nil {
		err := a.memory.Add(ctx, memory.MemoryItem{
			ID:        uuid.New().String(),
			Content:   a.currentThought,
			Type:      memory.TypeThought,
			CreatedAt: time.Now(),
			Metadata:  map[string]interface{}{"source": "llm"},
		})
		if err != nil {
			return fmt.Errorf("failed to add thought to memory: %w", err)
		}
	}

	return nil
}

// GetModelName returns the current model name being used
func (a *LLMAgent) GetModelName() string {
	// Here we need to get the model name from the Provider, since the interface might not provide it, returning a mock value
	// Ideally, this should be obtained from the Provider
	return "gpt-4o"
}

// Function to truncate string, used for truncating long content in log output
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Act executes decisions and returns results in the action phase
func (a *LLMAgent) Act(ctx context.Context) (interface{}, error) {
	// Check if thinking result contains tool call
	fmt.Println("========11111=========")
	fmt.Println(a.currentThought)
	fmt.Println("========11111=========")
	if isToolCall(a.currentThought) {
		log.Printf("Agent[%s] detected tool call intent, parsing tool call", a.Name())

		// Parse tool call
		toolName, params, err := parseToolCall(a.currentThought)
		if err != nil {
			log.Printf("Agent[%s] failed to parse tool call: %v", a.Name(), err)
			return nil, fmt.Errorf("failed to parse tool call: %w", err)
		}

		log.Printf("Agent[%s] successfully parsed tool call: tool=%s, params=%+v", a.Name(), toolName, params)

		// Call tool
		result, err := a.callTool(ctx, toolName, params)
		if err != nil {
			return nil, fmt.Errorf("tool call failed: %w", err)
		}

		// Add tool call result to memory
		if a.memory != nil {
			err := a.memory.Add(ctx, memory.MemoryItem{
				ID:        uuid.New().String(),
				Content:   result,
				Type:      memory.TypeAction,
				CreatedAt: time.Now(),
				Metadata: map[string]interface{}{
					"tool":   toolName,
					"params": params,
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to add tool call result to memory: %w", err)
			}
		}

		return result, nil
	}

	// If no tool call, return thinking result directly
	log.Printf("Agent[%s] did not detect tool call, returning thinking result directly", a.Name())
	return a.currentThought, nil
}

// Process executes the full perceive-think-act cycle
func (a *LLMAgent) Process(ctx context.Context, input interface{}) (interface{}, error) {
	if err := a.Perceive(ctx, input); err != nil {
		return nil, err
	}

	if err := a.Think(ctx); err != nil {
		return nil, err
	}

	return a.Act(ctx)
}

// preparePrompt prepares the prompt
func (a *LLMAgent) preparePrompt() string {
	var prompt string

	// Add system prompt
	prompt += a.systemPrompt + "\n\n"

	// Add current input
	if a.currentInput != nil {
		switch input := a.currentInput.(type) {
		case string:
			prompt += "Input: " + input + "\n\n"
		case map[string]interface{}:
			inputJSON, _ := json.MarshalIndent(input, "", "  ")
			prompt += "Input: " + string(inputJSON) + "\n\n"
		default:
			inputJSON, _ := json.MarshalIndent(input, "", "  ")
			prompt += "Input: " + string(inputJSON) + "\n\n"
		}
	}

	// Add available tools information
	if len(a.tools) > 0 {
		prompt += "Available tools:\n"
		for _, tool := range a.tools {
			prompt += fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description())
		}
		prompt += "\n"
	}

	// Add recent memories
	if a.memory != nil {
		ctx := context.Background() // Create a new context
		recentMemories, err := a.memory.GetRecent(ctx, 5)
		if err == nil && len(recentMemories) > 0 {
			prompt += "Recent memories:\n"
			for _, mem := range recentMemories {
				prompt += fmt.Sprintf("- [%s] %v\n", mem.Type, mem.Content)
			}
			prompt += "\n"
		}
	}

	// Add thinking instructions
	prompt += "Please analyze the above information and provide your analysis and decisions. If you need to use a tool, use the following format:\n"
	prompt += "Tool: <tool name>\n"
	prompt += "Parameters: <JSON formatted parameters>\n"

	return prompt
}

// isToolCall checks if text contains a tool call
func isToolCall(text string) bool {
	return strings.Contains(text, "Tool:") || strings.Contains(text, "tool:")
}

// parseToolCall parses a tool call
func parseToolCall(text string) (string, map[string]interface{}, error) {
	// Simple parsing logic, more complex parsing may be needed in practice
	var toolName string
	var paramsStr string

	// Find tool name
	toolRegex := regexp.MustCompile(`(?i)Tool:\s*(\w+)|tool:\s*(\w+)`)
	toolMatches := toolRegex.FindStringSubmatch(text)
	if len(toolMatches) > 1 {
		if toolMatches[1] != "" {
			toolName = toolMatches[1]
		} else if len(toolMatches) > 2 {
			toolName = toolMatches[2]
		}
	}

	// Find parameters
	paramsRegex := regexp.MustCompile(`(?i)Parameters:\s*(\{.*\})|params:\s*(\{.*\})`)
	paramsMatches := paramsRegex.FindStringSubmatch(text)
	if len(paramsMatches) > 1 {
		if paramsMatches[1] != "" {
			paramsStr = paramsMatches[1]
		} else if len(paramsMatches) > 2 {
			paramsStr = paramsMatches[2]
		}
	}

	if toolName == "" {
		return "", nil, fmt.Errorf("tool name not found")
	}

	// Parse parameters JSON
	var params map[string]interface{}
	if paramsStr != "" {
		log.Printf("Parsing tool parameter JSON: %s", paramsStr)
		if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
			return toolName, nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	} else {
		params = make(map[string]interface{})
		log.Printf("No tool parameters provided, using empty parameters")
	}

	return toolName, params, nil
}

// callTool calls a tool
func (a *LLMAgent) callTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// Find tool
	var selectedTool tools.Tool
	for _, tool := range a.tools {
		if tool.Name() == toolName {
			selectedTool = tool
			break
		}
	}

	if selectedTool == nil {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// Add tool call log
	log.Printf("Agent[%s] calling tool: %s, params: %+v", a.Name(), toolName, params)

	// Execute tool directly with parameter map
	result, err := selectedTool.Execute(ctx, params)
	if err != nil {
		log.Printf("Agent[%s] failed to call tool[%s]: %v", a.Name(), toolName, err)
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	// Add tool call result log
	log.Printf("Agent[%s] called tool[%s] successfully: %v", a.Name(), toolName, result)

	return result, nil
}
