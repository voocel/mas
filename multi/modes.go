package multi

import (
	"context"
	"fmt"
	"sync"

	"github.com/voocel/mas/agent"
	"github.com/voocel/mas/runner"
	"github.com/voocel/mas/schema"
)

// Reducer merges parallel results.
type Reducer func(results []schema.Message) (schema.Message, error)

// FirstReducer returns the first non-empty result.
func FirstReducer(results []schema.Message) (schema.Message, error) {
	if len(results) == 0 {
		return schema.Message{}, fmt.Errorf("multi: empty results")
	}
	for _, msg := range results {
		if msg.Content != "" || len(msg.ToolCalls) > 0 {
			return msg, nil
		}
	}
	return results[0], nil
}

// RunSequential executes agents in order.
func RunSequential(ctx context.Context, r *runner.Runner, agents []*agent.Agent, input schema.Message) (schema.Message, error) {
	if r == nil {
		return schema.Message{}, fmt.Errorf("multi: runner is nil")
	}
	if len(agents) == 0 {
		return schema.Message{}, fmt.Errorf("multi: agents is empty")
	}

	current := input
	var last schema.Message
	for _, ag := range agents {
		if ag == nil {
			return schema.Message{}, fmt.Errorf("multi: agent is nil")
		}
		run := r
		if r != nil && r.GetMemory() != nil {
			run = r.WithMemory(r.GetMemory().Clone())
		}
		resp, err := run.Run(ctx, ag, current)
		if err != nil {
			return schema.Message{}, err
		}
		last = resp
		current = resp
	}
	return last, nil
}

// RunParallel executes agents in parallel and merges results.
func RunParallel(ctx context.Context, r *runner.Runner, agents []*agent.Agent, input schema.Message, reducer Reducer) (schema.Message, error) {
	if r == nil {
		return schema.Message{}, fmt.Errorf("multi: runner is nil")
	}
	if len(agents) == 0 {
		return schema.Message{}, fmt.Errorf("multi: agents is empty")
	}
	if reducer == nil {
		reducer = FirstReducer
	}

	results := make([]schema.Message, len(agents))
	errs := make([]error, len(agents))

	var wg sync.WaitGroup
	for i, ag := range agents {
		wg.Add(1)
		go func(idx int, agent *agent.Agent) {
			defer wg.Done()
			if agent == nil {
				errs[idx] = fmt.Errorf("multi: agent is nil")
				return
			}
			run := r
			if r != nil && r.GetMemory() != nil {
				run = r.WithMemory(r.GetMemory().Clone())
			}
			resp, err := run.Run(ctx, agent, input)
			if err != nil {
				errs[idx] = err
				return
			}
			results[idx] = resp
		}(i, ag)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return schema.Message{}, err
		}
	}

	return reducer(results)
}

// HandoffOption configures handoff mode.
type HandoffOption func(*handoffConfig)

type handoffConfig struct {
	maxSteps int
}

// WithMaxSteps sets the maximum number of steps.
func WithMaxSteps(steps int) HandoffOption {
	return func(cfg *handoffConfig) {
		if steps > 0 {
			cfg.maxSteps = steps
		}
	}
}

// RunHandoff uses a Router to select agents step by step.
func RunHandoff(ctx context.Context, r *runner.Runner, team *Team, router Router, input schema.Message, opts ...HandoffOption) (schema.Message, error) {
	if r == nil {
		return schema.Message{}, fmt.Errorf("multi: runner is nil")
	}
	if team == nil {
		return schema.Message{}, fmt.Errorf("multi: team is nil")
	}
	if router == nil {
		return schema.Message{}, fmt.Errorf("multi: router is nil")
	}

	cfg := &handoffConfig{maxSteps: 3}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	current := input
	var last schema.Message
	run := r
	if r != nil && r.GetMemory() != nil {
		run = r.WithMemory(r.GetMemory().Clone())
	}
	for step := 0; step < cfg.maxSteps; step++ {
		ag, err := router.Select(current, team)
		if err != nil {
			return schema.Message{}, err
		}
		if ag == nil {
			return schema.Message{}, fmt.Errorf("multi: agent is nil")
		}
		resp, err := run.Run(ctx, ag, current)
		if err != nil {
			return schema.Message{}, err
		}
		last = resp
		current = resp
	}
	return last, nil
}
