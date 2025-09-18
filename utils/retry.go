package utils

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/voocel/mas/schema"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"` // Maximum retry attempts
	BaseDelay   time.Duration `json:"base_delay"`   // Base delay duration
	MaxDelay    time.Duration `json:"max_delay"`    // Maximum delay duration
	Multiplier  float64       `json:"multiplier"`   // Backoff multiplier
	Jitter      bool          `json:"jitter"`       // Whether to add random jitter
}

// DefaultRetryConfig provides default retry settings
var DefaultRetryConfig = &RetryConfig{
	MaxAttempts: 3,
	BaseDelay:   100 * time.Millisecond,
	MaxDelay:    5 * time.Second,
	Multiplier:  2.0,
	Jitter:      true,
}

// Execute runs the function with retry semantics
func (c *RetryConfig) Execute(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= c.MaxAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else if !schema.IsRetryable(err) {
			return err
		} else {
			lastErr = err
		}

		// Wait before retrying when more attempts remain
		if attempt < c.MaxAttempts {
			delay := c.calculateDelay(attempt)

			select {
			case <-time.After(delay):
				// Continue retrying
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// ExecuteWithResult retries the function and returns its result
func (c *RetryConfig) ExecuteWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	var result interface{}

	for attempt := 1; attempt <= c.MaxAttempts; attempt++ {
		// Invoke the function
		if res, err := fn(); err == nil {
			return res, nil
		} else if !schema.IsRetryable(err) {
			return nil, err
		} else {
			lastErr = err
			result = res
		}

		// Wait before retrying when more attempts remain
		if attempt < c.MaxAttempts {
			delay := c.calculateDelay(attempt)

			select {
			case <-time.After(delay):
				// Continue retrying
			case <-ctx.Done():
				return result, ctx.Err()
			}
		}
	}

	return result, lastErr
}

// calculateDelay determines the backoff delay
func (c *RetryConfig) calculateDelay(attempt int) time.Duration {
	// Exponential backoff
	delay := time.Duration(float64(c.BaseDelay) * math.Pow(c.Multiplier, float64(attempt-1)))

	// Clamp to the maximum delay
	if delay > c.MaxDelay {
		delay = c.MaxDelay
	}

	// Add optional jitter
	if c.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
		delay += jitter
	}

	return delay
}

// Retry executes with the default retry configuration
func Retry(ctx context.Context, fn func() error) error {
	return DefaultRetryConfig.Execute(ctx, fn)
}

// RetryWithResult executes with the default configuration and returns the result
func RetryWithResult(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return DefaultRetryConfig.ExecuteWithResult(ctx, fn)
}

// RetryWithConfig executes with a custom configuration
func RetryWithConfig(ctx context.Context, config *RetryConfig, fn func() error) error {
	if config == nil {
		config = DefaultRetryConfig
	}
	return config.Execute(ctx, fn)
}

// RetryWithConfigAndResult executes with a custom configuration and returns the result
func RetryWithConfigAndResult(ctx context.Context, config *RetryConfig, fn func() (interface{}, error)) (interface{}, error) {
	if config == nil {
		config = DefaultRetryConfig
	}
	return config.ExecuteWithResult(ctx, fn)
}

// NewRetryConfig constructs a retry configuration
func NewRetryConfig(maxAttempts int, baseDelay, maxDelay time.Duration, multiplier float64, jitter bool) *RetryConfig {
	return &RetryConfig{
		MaxAttempts: maxAttempts,
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		Multiplier:  multiplier,
		Jitter:      jitter,
	}
}

// LinearRetryConfig creates a linear (fixed-delay) configuration
func LinearRetryConfig(maxAttempts int, delay time.Duration, jitter bool) *RetryConfig {
	return &RetryConfig{
		MaxAttempts: maxAttempts,
		BaseDelay:   delay,
		MaxDelay:    delay,
		Multiplier:  1.0, // No growth
		Jitter:      jitter,
	}
}

// ExponentialRetryConfig creates an exponential backoff configuration
func ExponentialRetryConfig(maxAttempts int, baseDelay, maxDelay time.Duration, jitter bool) *RetryConfig {
	return &RetryConfig{
		MaxAttempts: maxAttempts,
		BaseDelay:   baseDelay,
		MaxDelay:    maxDelay,
		Multiplier:  2.0, // Exponential growth
		Jitter:      jitter,
	}
}

// RetryableFunc represents a retryable function
type RetryableFunc func() error

// RetryableFuncWithResult represents a retryable function that returns a value
type RetryableFuncWithResult func() (interface{}, error)

// RetryCondition decides whether to retry an error
type RetryCondition func(error) bool

// ConditionalRetry retries only when the condition allows
func ConditionalRetry(ctx context.Context, config *RetryConfig, condition RetryCondition, fn RetryableFunc) error {
	if config == nil {
		config = DefaultRetryConfig
	}

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else if !condition(err) {
			return err // Retry condition failed
		} else {
			lastErr = err
		}

		if attempt < config.MaxAttempts {
			delay := config.calculateDelay(attempt)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// RetryStats captures retry metrics
type RetryStats struct {
	TotalAttempts  int           `json:"total_attempts"`
	SuccessAttempt int           `json:"success_attempt"`
	TotalDuration  time.Duration `json:"total_duration"`
	LastError      error         `json:"last_error"`
}

// ExecuteWithStats retries and returns statistics
func (c *RetryConfig) ExecuteWithStats(ctx context.Context, fn func() error) (*RetryStats, error) {
	stats := &RetryStats{}
	startTime := time.Now()

	for attempt := 1; attempt <= c.MaxAttempts; attempt++ {
		stats.TotalAttempts = attempt

		if err := fn(); err == nil {
			stats.SuccessAttempt = attempt
			stats.TotalDuration = time.Since(startTime)
			return stats, nil
		} else if !schema.IsRetryable(err) {
			stats.LastError = err
			stats.TotalDuration = time.Since(startTime)
			return stats, err
		} else {
			stats.LastError = err
		}

		if attempt < c.MaxAttempts {
			delay := c.calculateDelay(attempt)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				stats.TotalDuration = time.Since(startTime)
				return stats, ctx.Err()
			}
		}
	}

	stats.TotalDuration = time.Since(startTime)
	return stats, stats.LastError
}
