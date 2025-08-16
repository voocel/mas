package workflow

import (
	"context"
	"fmt"
	"time"
)

// ConsoleInputProvider provides human input via console
type ConsoleInputProvider struct{}

// NewConsoleInputProvider creates a new console input provider
func NewConsoleInputProvider() *ConsoleInputProvider {
	return &ConsoleInputProvider{}
}

// RequestInput requests input from console with timeout
func (p *ConsoleInputProvider) RequestInput(ctx context.Context, prompt string, options ...HumanInputOption) (*HumanInput, error) {
	config := DefaultHumanInputConfig()
	for _, option := range options {
		option(&config)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	inputChan := make(chan *HumanInput, 1)
	errChan := make(chan error, 1)

	// Start input goroutine
	go func() {
		fmt.Printf("\nHuman Input Required:\n%s\n> ", prompt)

		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			errChan <- fmt.Errorf("failed to read input: %w", err)
			return
		}

		if config.Validator != nil {
			if err := config.Validator(input); err != nil {
				errChan <- fmt.Errorf("validation failed: %w", err)
				return
			}
		}

		if config.Required && input == "" {
			errChan <- fmt.Errorf("input is required")
			return
		}

		inputChan <- &HumanInput{
			Value: input,
			Data:  make(map[string]any),
		}
	}()

	select {
	case input := <-inputChan:
		return input, nil
	case err := <-errChan:
		return nil, err
	case <-timeoutCtx.Done():
		return nil, fmt.Errorf("human input timeout after %v", config.Timeout)
	}
}

// DefaultHumanInputConfig returns default configuration
func DefaultHumanInputConfig() HumanInputConfig {
	return HumanInputConfig{
		Timeout:  5 * time.Minute,
		Required: true,
	}
}

// Condition helpers for workflow routing

// WhenOutputContains creates a condition that checks if output contains a string
func WhenOutputContains(substring string) func(*WorkflowContext) bool {
	return func(ctx *WorkflowContext) bool {
		if output := ctx.Get("output"); output != nil {
			return fmt.Sprintf("%v", output) != "" && 
				   fmt.Sprintf("%v", output) != substring
		}
		return false
	}
}

// WhenOutputEquals creates a condition that checks if output equals a string
func WhenOutputEquals(expected string) func(*WorkflowContext) bool {
	return func(ctx *WorkflowContext) bool {
		if output := ctx.Get("output"); output != nil {
			return fmt.Sprintf("%v", output) == expected
		}
		return false
	}
}

// WhenDataExists creates a condition that checks if a data key exists
func WhenDataExists(key string) func(*WorkflowContext) bool {
	return func(ctx *WorkflowContext) bool {
		return ctx.Get(key) != nil
	}
}

// WhenDataEquals creates a condition that checks if a data value equals expected
func WhenDataEquals(key string, expected interface{}) func(*WorkflowContext) bool {
	return func(ctx *WorkflowContext) bool {
		value := ctx.Get(key)
		return value == expected
	}
}

// WhenTrue creates a condition that always returns true (for default cases)
func WhenTrue() func(*WorkflowContext) bool {
	return func(ctx *WorkflowContext) bool {
		return true
	}
}

// WhenFalse creates a condition that always returns false
func WhenFalse() func(*WorkflowContext) bool {
	return func(ctx *WorkflowContext) bool {
		return false
	}
}