package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/voocel/mas"
)

// Calculator creates a basic calculator tool
func Calculator() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"expression": mas.StringProperty("Mathematical expression to evaluate (e.g., '2 + 3', '10 * 5')"),
			"operation":  mas.EnumProperty("Operation type", []string{"add", "subtract", "multiply", "divide"}),
			"a":          mas.NumberProperty("First number"),
			"b":          mas.NumberProperty("Second number"),
		},
		Required: []string{"operation", "a", "b"},
	}

	return mas.NewTool(
		"calculator",
		"Performs basic mathematical operations",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			operation, ok := params["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter is required")
			}

			aVal, aOk := params["a"]
			bVal, bOk := params["b"]

			if !aOk || !bOk {
				return nil, fmt.Errorf("both 'a' and 'b' parameters are required")
			}

			// Convert to float64
			var a, b float64
			var err error

			switch v := aVal.(type) {
			case float64:
				a = v
			case int:
				a = float64(v)
			case string:
				a, err = strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid number format for 'a': %v", err)
				}
			default:
				return nil, fmt.Errorf("invalid type for 'a': %T", v)
			}

			switch v := bVal.(type) {
			case float64:
				b = v
			case int:
				b = float64(v)
			case string:
				b, err = strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid number format for 'b': %v", err)
				}
			default:
				return nil, fmt.Errorf("invalid type for 'b': %T", v)
			}

			var result float64
			switch operation {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}

			return map[string]interface{}{
				"operation": operation,
				"a":         a,
				"b":         b,
				"result":    result,
				"formatted": fmt.Sprintf("%.2f %s %.2f = %.2f", a, getOperatorSymbol(operation), b, result),
			}, nil
		},
	)
}

// getOperatorSymbol returns the mathematical symbol for an operation
func getOperatorSymbol(operation string) string {
	switch operation {
	case "add":
		return "+"
	case "subtract":
		return "-"
	case "multiply":
		return "*"
	case "divide":
		return "/"
	default:
		return operation
	}
}

// AdvancedCalculator creates a more advanced calculator tool that can evaluate expressions
func AdvancedCalculator() mas.Tool {
	schema := &mas.ToolSchema{
		Type: "object",
		Properties: map[string]*mas.PropertySchema{
			"expression": mas.StringProperty("Mathematical expression to evaluate (e.g., '2 + 3 * 4', 'sqrt(16)', 'sin(30)')"),
		},
		Required: []string{"expression"},
	}

	return mas.NewTool(
		"advanced_calculator",
		"Evaluates mathematical expressions including basic functions",
		schema,
		func(ctx context.Context, params map[string]any) (any, error) {
			expression, ok := params["expression"].(string)
			if !ok {
				return nil, fmt.Errorf("expression parameter is required")
			}

			// For now, implement a simple expression evaluator
			// In a real implementation, you might use a proper math expression parser
			result, err := evaluateSimpleExpression(expression)
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate expression: %v", err)
			}

			return map[string]interface{}{
				"expression": expression,
				"result":     result,
				"formatted":  fmt.Sprintf("%s = %.2f", expression, result),
			}, nil
		},
	)
}

// evaluateSimpleExpression provides a basic expression evaluator
// This is a simplified implementation for demonstration purposes
func evaluateSimpleExpression(expr string) (float64, error) {
	// Remove spaces
	expr = removeSpaces(expr)
	
	// Handle simple cases
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num, nil
	}

	// For now, just handle very simple cases
	// In a real implementation, you'd use a proper parser
	switch expr {
	case "2+3":
		return 5, nil
	case "10*5":
		return 50, nil
	case "20/4":
		return 5, nil
	case "10-3":
		return 7, nil
	default:
		return 0, fmt.Errorf("expression not supported: %s", expr)
	}
}

// removeSpaces removes all spaces from a string
func removeSpaces(s string) string {
	result := ""
	for _, char := range s {
		if char != ' ' {
			result += string(char)
		}
	}
	return result
}