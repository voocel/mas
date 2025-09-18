package builtin

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/voocel/mas/runtime"
	"github.com/voocel/mas/schema"
	"github.com/voocel/mas/tools"
)

// Calculator evaluates arithmetic expressions
type Calculator struct {
	*tools.BaseTool
}

// NewCalculator constructs the calculator tool
func NewCalculator() *Calculator {
	schema := tools.CreateToolSchema(
		"Perform basic mathematical calculations",
		map[string]interface{}{
			"expression": tools.StringProperty("Mathematical expression to evaluate (e.g., '2 + 3 * 4')"),
		},
		[]string{"expression"},
	)
	
	baseTool := tools.NewBaseTool("calculator", "Perform basic mathematical calculations", schema)
	
	return &Calculator{
		BaseTool: baseTool,
	}
}

// Execute evaluates the expression
func (c *Calculator) Execute(ctx runtime.Context, input json.RawMessage) (json.RawMessage, error) {
	// Parse input parameters
	var params struct {
		Expression string `json:"expression"`
	}
	
	if err := json.Unmarshal(input, &params); err != nil {
		return nil, schema.NewValidationError("input", string(input), "invalid JSON format")
	}
	
	if params.Expression == "" {
		return nil, schema.NewValidationError("expression", params.Expression, "expression cannot be empty")
	}
	
	// Normalize the expression
	expression := strings.TrimSpace(params.Expression)
	
	// Evaluate the expression
	result, err := c.evaluate(expression)
	if err != nil {
		return nil, schema.NewToolError(c.Name(), "evaluate", err)
	}
	
	// Return the result
	response := map[string]interface{}{
		"expression": expression,
		"result":     result,
	}
	
	return json.Marshal(response)
}

// evaluate evaluates an expression
func (c *Calculator) evaluate(expr string) (float64, error) {
	// Use Go's AST parser to evaluate the expression safely
	node, err := parser.ParseExpr(expr)
	if err != nil {
		return 0, fmt.Errorf("invalid expression: %v", err)
	}
	
	return c.evalNode(node)
}

// evalNode recursively evaluates AST nodes
func (c *Calculator) evalNode(node ast.Node) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return c.evalBasicLit(n)
	case *ast.BinaryExpr:
		return c.evalBinaryExpr(n)
	case *ast.UnaryExpr:
		return c.evalUnaryExpr(n)
	case *ast.ParenExpr:
		return c.evalNode(n.X)
	default:
		return 0, fmt.Errorf("unsupported expression type: %T", n)
	}
}

// evalBasicLit evaluates literals
func (c *Calculator) evalBasicLit(lit *ast.BasicLit) (float64, error) {
	switch lit.Kind {
	case token.INT:
		val, err := strconv.ParseInt(lit.Value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid integer: %s", lit.Value)
		}
		return float64(val), nil
	case token.FLOAT:
		val, err := strconv.ParseFloat(lit.Value, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid float: %s", lit.Value)
		}
		return val, nil
	default:
		return 0, fmt.Errorf("unsupported literal type: %s", lit.Kind)
	}
}

// evalBinaryExpr evaluates binary expressions
func (c *Calculator) evalBinaryExpr(expr *ast.BinaryExpr) (float64, error) {
	left, err := c.evalNode(expr.X)
	if err != nil {
		return 0, err
	}
	
	right, err := c.evalNode(expr.Y)
	if err != nil {
		return 0, err
	}
	
	switch expr.Op {
	case token.ADD:
		return left + right, nil
	case token.SUB:
		return left - right, nil
	case token.MUL:
		return left * right, nil
	case token.QUO:
		if right == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return left / right, nil
	case token.REM:
		if right == 0 {
			return 0, fmt.Errorf("modulo by zero")
		}
		return float64(int64(left) % int64(right)), nil
	default:
		return 0, fmt.Errorf("unsupported binary operator: %s", expr.Op)
	}
}

// evalUnaryExpr evaluates unary expressions
func (c *Calculator) evalUnaryExpr(expr *ast.UnaryExpr) (float64, error) {
	operand, err := c.evalNode(expr.X)
	if err != nil {
		return 0, err
	}
	
	switch expr.Op {
	case token.ADD:
		return operand, nil
	case token.SUB:
		return -operand, nil
	default:
		return 0, fmt.Errorf("unsupported unary operator: %s", expr.Op)
	}
}

// GetSupportedOperations lists supported operations
func (c *Calculator) GetSupportedOperations() []string {
	return []string{
		"Addition (+)",
		"Subtraction (-)",
		"Multiplication (*)",
		"Division (/)",
		"Modulo (%)",
		"Parentheses for grouping",
		"Positive/Negative numbers",
	}
}

// ValidateExpression ensures the expression is safe
func (c *Calculator) ValidateExpression(expr string) error {
	// Enforce expression length limits
	if len(expr) > 1000 {
		return fmt.Errorf("expression too long (max 1000 characters)")
	}
	
	// Check for unsafe keywords
	unsafeChars := []string{"import", "func", "var", "const", "package", "go", "chan", "select"}
	lowerExpr := strings.ToLower(expr)
	
	for _, unsafe := range unsafeChars {
		if strings.Contains(lowerExpr, unsafe) {
			return fmt.Errorf("expression contains unsafe keyword: %s", unsafe)
		}
	}
	
	// Attempt to parse the expression
	_, err := parser.ParseExpr(expr)
	if err != nil {
		return fmt.Errorf("invalid expression syntax: %v", err)
	}
	
	return nil
}

// Examples returns usage samples
func (c *Calculator) Examples() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"description": "Simple addition",
			"input":       map[string]string{"expression": "2 + 3"},
			"output":      map[string]interface{}{"expression": "2 + 3", "result": 5.0},
		},
		{
			"description": "Complex expression with parentheses",
			"input":       map[string]string{"expression": "(10 + 5) * 2 - 3"},
			"output":      map[string]interface{}{"expression": "(10 + 5) * 2 - 3", "result": 27.0},
		},
		{
			"description": "Division with decimal result",
			"input":       map[string]string{"expression": "7 / 2"},
			"output":      map[string]interface{}{"expression": "7 / 2", "result": 3.5},
		},
		{
			"description": "Negative numbers",
			"input":       map[string]string{"expression": "-5 + 10"},
			"output":      map[string]interface{}{"expression": "-5 + 10", "result": 5.0},
		},
	}
}
