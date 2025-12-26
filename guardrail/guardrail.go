package guardrail

import (
	"context"
	"fmt"

	"github.com/voocel/mas/schema"
)

// Result represents the outcome of a guardrail check.
type Result struct {
	Passed  bool
	Reason  string
	Details map[string]interface{}
}

// Pass returns a successful guardrail result.
func Pass() Result {
	return Result{Passed: true}
}

// Block returns a failed guardrail result with a reason.
func Block(reason string) Result {
	return Result{Passed: false, Reason: reason}
}

// BlockWithDetails returns a failed guardrail result with additional details.
func BlockWithDetails(reason string, details map[string]interface{}) Result {
	return Result{Passed: false, Reason: reason, Details: details}
}

// InputGuardrail validates input before LLM processing.
type InputGuardrail interface {
	Name() string
	ValidateInput(ctx context.Context, input *schema.Message) Result
}

// OutputGuardrail validates output after LLM processing.
type OutputGuardrail interface {
	Name() string
	ValidateOutput(ctx context.Context, output *schema.Message) Result
}

// GuardrailError represents a guardrail validation failure.
type GuardrailError struct {
	GuardrailName string
	Type          string // "input" or "output"
	Reason        string
	Details       map[string]interface{}
}

func (e *GuardrailError) Error() string {
	return fmt.Sprintf("guardrail %q (%s) blocked: %s", e.GuardrailName, e.Type, e.Reason)
}

// ValidateFunc is the common validation function signature.
type ValidateFunc func(ctx context.Context, msg *schema.Message) Result

// inputFunc wraps a function as InputGuardrail.
type inputFunc struct {
	name string
	fn   ValidateFunc
}

func (g *inputFunc) Name() string { return g.name }
func (g *inputFunc) ValidateInput(ctx context.Context, input *schema.Message) Result {
	return g.fn(ctx, input)
}

// outputFunc wraps a function as OutputGuardrail.
type outputFunc struct {
	name string
	fn   ValidateFunc
}

func (g *outputFunc) Name() string { return g.name }
func (g *outputFunc) ValidateOutput(ctx context.Context, output *schema.Message) Result {
	return g.fn(ctx, output)
}

// NewInputGuardrail creates an input guardrail from a function.
func NewInputGuardrail(name string, fn ValidateFunc) InputGuardrail {
	return &inputFunc{name: name, fn: fn}
}

// NewOutputGuardrail creates an output guardrail from a function.
func NewOutputGuardrail(name string, fn ValidateFunc) OutputGuardrail {
	return &outputFunc{name: name, fn: fn}
}

// InputChain combines multiple input guardrails.
func InputChain(name string, guardrails ...InputGuardrail) InputGuardrail {
	return NewInputGuardrail(name, func(ctx context.Context, msg *schema.Message) Result {
		for _, g := range guardrails {
			if result := g.ValidateInput(ctx, msg); !result.Passed {
				return result
			}
		}
		return Pass()
	})
}

// OutputChain combines multiple output guardrails.
func OutputChain(name string, guardrails ...OutputGuardrail) OutputGuardrail {
	return NewOutputGuardrail(name, func(ctx context.Context, msg *schema.Message) Result {
		for _, g := range guardrails {
			if result := g.ValidateOutput(ctx, msg); !result.Passed {
				return result
			}
		}
		return Pass()
	})
}
