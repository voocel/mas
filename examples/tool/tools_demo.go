package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/llm"
	"github.com/voocel/mas/tools"
)

// ToolsDemo demonstrates how to use the refactored tools system
func main() {
	// Initialize LLM provider
	provider, err := llm.NewOpenAIProvider(llm.Config{
		APIKey:       os.Getenv("OPENAI_API_KEY"),
		DefaultModel: "gpt-4o",
		Timeout:      30,
	})
	if err != nil {
		log.Fatalf("Failed to initialize LLM provider: %v", err)
	}

	// Create toolbox
	toolbox := tools.NewToolbox()

	// Add HTTP tool
	httpTool := tools.NewHTTPTool()
	toolbox.Add(httpTool)

	// Add custom tool
	timeTool := tools.NewTool(
		"current_time",
		"Returns the current system time",
		tools.NewRawSchema(`{
			"type": "object",
			"properties": {},
			"additionalProperties": false
		}`),
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return time.Now().Format("2006-01-02 15:04:05"), nil
		},
	)
	toolbox.Add(timeTool)

	// Create an adaptive tool
	calculatorFunc := func(a, b float64, op string) (float64, error) {
		switch op {
		case "+":
			return a + b, nil
		case "-":
			return a - b, nil
		case "*":
			return a * b, nil
		case "/":
			if b == 0 {
				return 0, fmt.Errorf("Division by zero is not allowed")
			}
			return a / b, nil
		default:
			return 0, fmt.Errorf("Unsupported operation: %s", op)
		}
	}

	// Use adapter to convert function to tool
	calcTool := tools.NewToolAdapter(
		"calculator",
		"Performs basic mathematical operations, can use a, b, operation parameters or directly use the expression parameter",
		tools.NewRawSchema(`{
			"type": "object",
			"properties": {
				"expression": {
					"type": "string",
					"description": "Mathematical expression, such as '15 + 27'"
				},
				"a": {
					"type": "number",
					"description": "First operand"
				},
				"b": {
					"type": "number",
					"description": "Second operand"
				},
				"operation": {
					"type": "string",
					"enum": ["+", "-", "*", "/"],
					"description": "Operator"
				}
			},
			"oneOf": [
				{
					"required": ["expression"]
				},
				{
					"required": ["a", "b", "operation"]
				}
			]
		}`),
		func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			// Try to parse expression parameter
			if expr, ok := params["expression"].(string); ok {
				log.Printf("Received expression calculation request: %s", expr)

				// Simple expression parsing, supports format like "15 + 27"
				parts := strings.Fields(expr)
				if len(parts) != 3 {
					return nil, fmt.Errorf("Expression format error, needs to be like '15 + 27'")
				}

				a, err1 := strconv.ParseFloat(parts[0], 64)
				b, err2 := strconv.ParseFloat(parts[2], 64)
				op := parts[1]

				if err1 != nil || err2 != nil {
					return nil, fmt.Errorf("Cannot parse numbers: %v, %v", err1, err2)
				}

				result, err := calculatorFunc(a, b, op)
				if err != nil {
					return nil, err
				}

				return map[string]interface{}{
					"result":     result,
					"a":          a,
					"b":          b,
					"op":         op,
					"expression": expr,
				}, nil
			}

			// Traditional way: using a, b, operation parameters
			a, ok1 := params["a"].(float64)
			b, ok2 := params["b"].(float64)
			op, ok3 := params["operation"].(string)

			if !ok1 || !ok2 || !ok3 {
				return nil, fmt.Errorf("Parameter error: need a (number), b (number) and operation (string), or use the expression parameter")
			}

			result, err := calculatorFunc(a, b, op)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{
				"result": result,
				"a":      a,
				"b":      b,
				"op":     op,
			}, nil
		},
	)
	toolbox.Add(calcTool)

	// Create agent
	assistant := agent.NewLLMAgent(agent.LLMAgentConfig{
		Name:         "Tool Assistant",
		Provider:     provider,
		SystemPrompt: "You are a helpful assistant that can use various tools to complete tasks.",
		Tools:        toolbox.List(),
		MaxTokens:    1000,
		Temperature:  0.7,
	})

	// Process user requests
	fmt.Println("===== Tool Assistant Example =====")
	fmt.Println("Available tools:", toolbox.Names())

	// Test LLM agent's ability to use tools
	queries := []string{
		"What time is it now?",
		"Please calculate the result of 15 + 27",
		"Please get the content of https://jsonplaceholder.typicode.com/todos/1",
	}

	for _, query := range queries {
		fmt.Printf("\n> %s\n", query)
		result, err := assistant.Process(context.Background(), query)
		if err != nil {
			fmt.Printf("Processing failed: %v\n", err)
			continue
		}
		fmt.Printf("Answer: %v\n", result)
	}

	fmt.Println("\n===== Direct Tool Call Example =====")

	// Direct tool call examples
	timeResult, err := toolbox.Execute(context.Background(), "current_time", map[string]interface{}{})
	if err != nil {
		fmt.Printf("Failed to call time tool: %v\n", err)
	} else {
		fmt.Printf("Current time: %v\n", timeResult)
	}

	calcResult, err := toolbox.Execute(context.Background(), "calculator", map[string]interface{}{
		"a":         10.5,
		"b":         2.0,
		"operation": "*",
	})
	if err != nil {
		fmt.Printf("Failed to call calculator tool: %v\n", err)
	} else {
		fmt.Printf("Calculation result: %v\n", calcResult)
	}
}
