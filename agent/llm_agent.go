package agent

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Call LLM for thinking
	resp, err := a.provider.ChatCompletion(ctx, llm.ChatCompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.systemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: a.temperature,
		MaxTokens:   a.maxTokens,
	})
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract thinking result
	if len(resp.Choices) > 0 {
		a.currentThought = resp.Choices[0].Message.Content
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

// Act executes decisions and returns results in the action phase
func (a *LLMAgent) Act(ctx context.Context) (interface{}, error) {
	// Check if thinking result contains tool call
	if isToolCall(a.currentThought) {
		// Parse tool call
		toolName, params, err := parseToolCall(a.currentThought)
		if err != nil {
			return nil, fmt.Errorf("failed to parse tool call: %w", err)
		}

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
return a.currentThought, nil
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
		if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
			return toolName, nil, fmt.Errorf("failed to parse parameters: %w", err)
		}
	} else {
		params = make(map[string]interface{})
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

	// Convert parameters to JSON
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize parameters: %w", err)
	}

	// Execute tool
	result, err := selectedTool.Execute(ctx, paramsJSON)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return result, nil
}
